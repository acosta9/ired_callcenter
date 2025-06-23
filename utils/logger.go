package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"runtime"

	"ired.com/callcenter/models"
)

func Fatalf(format string, args ...interface{}) {
	// Get the file and line number
	_, file, line, _ := runtime.Caller(1)
	newFormat := fmt.Sprintf("%s:%d: %s", file, line, format)
	log.Fatalf(newFormat, args...)
}

func Logline(format string, args ...any) {
	// Get the file and line number
	_, file, line, _ := runtime.Caller(1)
	newFormat := fmt.Sprintf("%s:%d: %s", file, line, format)
	log.Println(newFormat, args)
}

// show status of db connections pgsql
func ShowStatusWorker(db models.ConnDb, taskName string, taskStatus string) {
	// get stats from pgsql pool
	statsPgsql := db.Conn.Stat()

	// transform to json format
	poolStats := models.WorkerStatus{
		TaskName:   taskName,
		TaskStatus: taskStatus,
		PgsqlStatus: models.PoolStatsPgsql{
			AcquiredConns:        statsPgsql.AcquiredConns(),
			TotalConns:           statsPgsql.TotalConns(),
			IdleConns:            statsPgsql.IdleConns(),
			MaxConns:             statsPgsql.MaxConns(),
			AcquireCount:         statsPgsql.AcquireCount(),
			AcquireDuration:      statsPgsql.AcquireDuration().String(),
			CanceledAcquireCount: statsPgsql.CanceledAcquireCount(),
			ConstructingConns:    statsPgsql.ConstructingConns(),
			EmptyAcquireCount:    statsPgsql.EmptyAcquireCount(),
		},
	}
	jsonData, err := json.Marshal(poolStats)
	if err != nil {
		Logline("Error parsing the json poolStat" + err.Error())
	}

	// print to logfile
	Logline(string(jsonData))
}
