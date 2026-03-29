package rdb

import (
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"

	"github.com/hsuanshao/golibs/ctx"
)

// routeMode is the enum for connection pool selection approach
type routeMode int

const (
	routeModeAuto     routeMode = 1
	routeModeLeader   routeMode = 2
	routeModeFollower routeMode = 3
)

type impl struct {
	Primary      *pgxpool.Pool   // Connect to HAProxy Primary
	Replicas     []*pgxpool.Pool // Connect to HAProxy Replica
	replicaIndex uint64
	mode         routeMode
}

// NewRDBClient is the constructor for DBClient
func NewRDBClient(ctx ctx.CTX, conf *Config) (client DBClient, err error) {
	if conf == nil {
		ctx.Error("input config for NewRDBClient with nil input")
		return nil, DBErrInvalidConfig
	}

	if conf.WriteConnections < 1 {
		conf.WriteConnections = 5
	}

	if conf.ReadConnections < 1 {
		conf.ReadConnections = 10
	}

	// For both Master and Replica min connection setup
	minConns := int32(2)

	// For both Master and Replica healthy check
	healthyCheckPeriod := 30 * time.Second

	mConfig, err := pgxpool.ParseConfig(conf.Master)
	if err != nil {
		ctx.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to parse write DSN")
		return nil, err
	}
	mConfig.MaxConns = conf.WriteConnections
	mConfig.MinConns = minConns
	mConfig.HealthCheckPeriod = healthyCheckPeriod

	var replicaConfs []*pgxpool.Config
	// Replicas
	if len(conf.Replicas) == 0 {
		replicaConfs = make([]*pgxpool.Config, 1)
		replicaConfs[0] = mConfig
	}

	if len(conf.Replicas) > 0 {
		replicaConfs = make([]*pgxpool.Config, len(conf.Replicas))
		for i, replicaDSN := range conf.Replicas {
			replicaConfs[i], err = pgxpool.ParseConfig(replicaDSN)
			replicaConfs[i].MaxConns = conf.ReadConnections
			replicaConfs[i].MinConns = minConns
			replicaConfs[i].HealthCheckPeriod = healthyCheckPeriod
			if err != nil {
				ctx.WithFields(logrus.Fields{
					"error": err,
				}).Error("Failed to parse read DSN")
				return nil, err
			}
		}
	}

	primary, err := pgxpool.NewWithConfig(ctx.Context, mConfig)
	if err != nil {
		return nil, err
	}

	replicaPools := make([]*pgxpool.Pool, len(conf.Replicas))
	for i, replicaConf := range replicaConfs {
		replicaPools[i], err = pgxpool.NewWithConfig(ctx.Context, replicaConf)
		if err != nil {
			return nil, err
		}
	}

	return &impl{Primary: primary, Replicas: replicaPools}, nil
}

// Encapsulated method: automatic routing
func (c *impl) Exec(ctx ctx.CTX, sql string, args ...any) (pgconn.CommandTag, DBError) {
	// Write operations are forced to go to Primary
	ctg, err := c.Primary.Exec(ctx, sql, args...)
	if err != nil {
		ctx.WithFields(logrus.Fields{"err": err, "sql": sql, "args": args}).Error("execute sql with args get error from pgx")
		return pgconn.CommandTag{}, DBErrExecuteQueryStmtFailed
	}

	return ctg, nil
}

// pickPool provides control the connection pool selection approach
func (c *impl) pickPool(isWrite bool) *pgxpool.Pool {
	if c.mode == routeModeLeader || isWrite || len(c.Replicas) == 0 {
		return c.Primary
	}
	// idx is the round robin index for replicas to avoid hot spot impact
	idx := atomic.AddUint64(&c.replicaIndex, 1) % uint64(len(c.Replicas))
	replicaNum := len(c.Replicas)
	newIdx := int(idx % uint64(len(c.Replicas)))
	if (replicaNum - 1) < (newIdx) {
		newIdx = replicaNum - 1
	}
	return c.Replicas[newIdx]
}

func (c *impl) QueryRow(ctx ctx.CTX, sql string, args ...any) pgx.Row {
	// Query operations go to Replica

	pgxRow := c.pickPool(false).QueryRow(ctx.Context, sql, args...)

	return pgxRow
}

func (c *impl) Query(ctx ctx.CTX, sql string, args ...any) (pgx.Rows, DBError) {
	// Query operations go to Replica

	pgxRows, err := c.pickPool(false).Query(ctx.Context, sql, args...)
	if err != nil {
		ctx.WithFields(logrus.Fields{"err": err, "sql": sql, "args": args}).Error("execute sql with args get error from pgx")
		return nil, DBErrExecuteQueryStmtFailed
	}
	return pgxRows, nil
}

// WithTransaction handle DB Transaction from begin, commit and rollback
func (c *impl) WithTransaction(ctx ctx.CTX, fn TxBlock) DBError {
	tx, err := c.Primary.Begin(ctx.Context)
	if err != nil {
		ctx.WithField("error", err).Error("Failed to begin transaction")
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback(ctx.Context)
			ctx.WithField("error", p).Error("Transaction failed")
			panic(p)
		}
	}()

	if err := fn(ctx, tx); err != nil {
		ctx.WithField("error", err).Error("Transaction failed")
		return DBErrCommitTxFailed
	}

	if err := tx.Commit(ctx.Context); err != nil {
		ctx.WithField("error", err).Error("Failed to commit transaction")
		return err
	}

	return nil
}

func (c *impl) Stats(ctx ctx.CTX) DBStats {
	mStat := c.Primary.Stat()
	/**
	Connection ping latency
	*/
	masterConnectStat := true
	masterPingErr := c.Primary.Ping(ctx.Context)
	if masterPingErr != nil {
		masterConnectStat = false
	}

	stats := DBStats{
		Master: MasterStats{
			PoolStats: PoolStats{
				TotalConns:             mStat.TotalConns(),
				IdleConns:              mStat.IdleConns(),
				ActiveConns:            mStat.TotalConns() - mStat.IdleConns(),
				AcquireCount:           mStat.AcquireCount(),
				AcquireDurationSeconds: mStat.AcquireDuration().Seconds(),
				ConnectStat:            masterConnectStat,
			},
		},
		Replicas: make([]ReplicaStats, 0, len(c.Replicas)),
	}

	// get all Replicas stats
	for i, rPool := range c.Replicas {
		rStat := rPool.Stat()
		rConnectStat := true
		rPingErr := rPool.Ping(ctx.Context)
		if rPingErr != nil {
			rConnectStat = false
		}
		stats.Replicas = append(stats.Replicas, ReplicaStats{
			ID: i,
			PoolStats: PoolStats{
				TotalConns:             rStat.TotalConns(),
				IdleConns:              rStat.IdleConns(),
				ActiveConns:            rStat.TotalConns() - rStat.IdleConns(),
				AcquireCount:           rStat.AcquireCount(),
				AcquireDurationSeconds: rStat.AcquireDuration().Seconds(),
				ConnectStat:            rConnectStat,
			},
		})
	}

	return stats
}

func (c *impl) Leader() DBClient {
	return &impl{
		Primary:      c.Primary,
		Replicas:     c.Replicas,
		mode:         routeModeLeader,
		replicaIndex: c.replicaIndex,
	}
}

func (c *impl) Follower() DBClient {
	return &impl{
		Primary:      c.Primary,
		Replicas:     c.Replicas,
		mode:         routeModeFollower,
		replicaIndex: c.replicaIndex,
	}
}

func (c *impl) Close() {
	c.Primary.Close()
	for _, replica := range c.Replicas {
		replica.Close()
	}
}

func (c *impl) NewBatchWriter(ctx ctx.CTX) BatchWriter {
	return &batchImpl{
		conn: c.Primary,
	}
}

type batchImpl struct {
	conn    *pgxpool.Pool
	table   string
	columns []string
	rows    [][]any
}

const autoFlushThreshold = 2000

// Example: Prepare(ctx, "sensors", []string{"ts", "device_id", "val"})
func (b *batchImpl) Prepare(ctx ctx.CTX, tableName string, columns []string) DBError {
	b.table = tableName
	b.columns = columns
	return nil
}

func (b *batchImpl) Append(c ctx.CTX, args ...any) DBError {
	if len(args) != len(b.columns) {
		c.WithFields(logrus.Fields{"expected": len(b.columns), "actual": len(args)}).Error("BatchWriter error: column count mismatch")
		return DBErrInvalidArgsNum
	}
	b.rows = append(b.rows, args)
	if len(b.rows) >= autoFlushThreshold {
		return b.Flush(ctx.TODO(c))
	}
	return nil
}

func (b *batchImpl) Flush(ctx ctx.CTX) DBError {
	if len(b.rows) == 0 {
		return nil
	}

	// core: use pgx CopyFrom to implement efficient writing
	count, err := b.conn.CopyFrom(
		ctx.Context,
		pgx.Identifier{b.table},
		b.columns,
		pgx.CopyFromRows(b.rows),
	)

	ctx.WithField("count", count).Debug("Flush copy from primary connection counts")

	if err != nil {
		ctx.WithField("error", err).Error("Failed to flush copy from")
		return DBErrFlushCopyFromFailed
	}

	// empty cache rows
	b.rows = b.rows[:0]
	return nil
}

func (b *batchImpl) Close() DBError {
	return nil
}
