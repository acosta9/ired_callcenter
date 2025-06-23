package models

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ErrorResponse struct {
	Error interface{} `json:"error"`
}

type SuccessResponse struct {
	Notice string      `json:"notice,omitempty"`
	Record interface{} `json:"record,omitempty"`
}

type ConnDb struct {
	Conn *pgxpool.Pool
	Ctx  context.Context
}

type WorkerStatus struct {
	TaskName    string `json:"task"`
	TaskStatus  string `json:"status"`
	PgsqlStatus PoolStatsPgsql
}

type PoolStatsPgsql struct {
	AcquiredConns        int32  `json:"acquired_conns"`
	TotalConns           int32  `json:"total_conns"`
	IdleConns            int32  `json:"idle_conns"`
	MaxConns             int32  `json:"max_conns"`
	AcquireCount         int64  `json:"acquire_count"`
	AcquireDuration      string `json:"acquire_duration"`
	CanceledAcquireCount int64  `json:"canceled_acquire_count"`
	ConstructingConns    int32  `json:"constructing_conns"`
	EmptyAcquireCount    int64  `json:"empty_acquire_count"`
}

type ConnMysql struct {
	Conn *sql.DB
	Ctx  context.Context
}
