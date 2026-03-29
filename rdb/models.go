package rdb

import (
	"errors"
)

// Config holds the configuration parameters for the datastore connections
type Config struct {
	Master           string   `json:"master_dsn"`
	Replicas         []string `json:"replicas_dsns"`
	WriteConnections int32    `json:"write_connections"`
	ReadConnections  int32    `json:"read_connections"`
}

type DBError error

var (
	DBErrMasterNotFound    DBError = errors.New("master not found")
	DBErrReplicaNotFound   DBError = errors.New("replica not found")
	DBErrInvalidConfig     DBError = errors.New("invalid config")
	DBErrConnectionFailed  DBError = errors.New("connection failed")
	DBErrHealthCheckFailed DBError = errors.New("health check failed")
	DBErrUnknown           DBError = errors.New("unknown error")

	DBErrFlushCopyFromFailed DBError = errors.New("flush copy from primary connection failed")
	DBErrInvalidArgsNum      DBError = errors.New("number of arguments does not match number of columns")

	DBErrExecuteQueryStmtFailed DBError = errors.New("execute query statement failed")
	DBErrCommitTxFailed         DBError = errors.New("commit transaction failed")
)
