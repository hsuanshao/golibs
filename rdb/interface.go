package rdb

import (
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/hsuanshao/golibs/ctx"
)

type TxBlock func(ctx ctx.CTX, tx pgx.Tx) error

// DBClient interface defines the methods we use from pgxpool.Pool
type DBClient interface {
	Exec(ctx ctx.CTX, sql string, args ...any) (pgconn.CommandTag, DBError)
	Query(ctx ctx.CTX, sql string, args ...any) (pgx.Rows, DBError)
	QueryRow(ctx ctx.CTX, sql string, args ...any) pgx.Row
	// WithTransaction automatic process Begin, Commit, Rollback and Recovery
	WithTransaction(ctx ctx.CTX, fn TxBlock) DBError
	NewBatchWriter(ctx ctx.CTX) BatchWriter

	// Leader and Follower for force routing switch
	Leader() DBClient
	Follower() DBClient
	Stats(ctx ctx.CTX) DBStats
	Close()
}

// BatchWriter interface defines the methods we handle batch write to database process
type BatchWriter interface {
	// Prepare specifies the target table and columns
	// Example: Prepare(ctx, "sensors", []string{"ts", "device_id", "val"})
	Prepare(ctx ctx.CTX, tableName string, columns []string) DBError

	// Append adds a record to the Buffer, does not involve network IO
	Append(ctx ctx.CTX, args ...any) DBError

	// Flush writes the data in the Buffer to the Master at once via the COPY protocol
	Flush(ctx ctx.CTX) DBError

	// Close releases resources
	Close() DBError
}

type DBStats struct {
	// Master status
	Master MasterStats `json:"master"`
	// Replicas status (Slice index corresponds to Replica order)
	Replicas []ReplicaStats `json:"replicas"`
}

type PoolStats struct {
	TotalConns             int32   // Total current connections
	IdleConns              int32   // Idle connections
	ActiveConns            int32   // Connections currently executing tasks
	AcquireCount           int64   // Cumulative connection request count
	AcquireDurationSeconds float64 // Cumulative total duration waiting for connection (used to determine resource bottlenecks)
	ConnectStat            bool    // Connection status
}

type MasterStats struct {
	PoolStats
}

type ReplicaStats struct {
	PoolStats
	ID int // Replica index, for correspondence in monitoring
}
