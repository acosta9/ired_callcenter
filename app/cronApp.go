package app

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/go-co-op/gocron/v2"
	"ired.com/callcenter/models"
	"ired.com/callcenter/repo"
	"ired.com/callcenter/utils"
)

// TaskConfig structure to hold the cron schedule and task name
type taskConfig struct {
	Schedule string `json:"schedule"`
	Task     string `json:"task"`
	Enabled  bool   `json:"enabled"`
}

// Load task configurations from file
func loadTasksConfig() ([]taskConfig, error) {
	// open file
	file, err := os.Open(".crontab")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// decode json data to struct
	var tasksConfig []taskConfig
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&tasksConfig)
	if err != nil {
		return nil, err
	}

	return tasksConfig, nil
}

func LoadCrontab() {

	// Load task configurations
	tasksConfig, err := loadTasksConfig()
	if err != nil {
		utils.Logline("Failed to load task configurations: %v", err)
		return
	}

	// Use America/Caracas time
	ccsLocation, _ := time.LoadLocation("America/Caracas")

	// Create a new scheduler
	scheduler, _ := gocron.NewScheduler(gocron.WithLocation(ccsLocation))

	// // Schedule tasks based on the configurations
	for _, taskConfig := range tasksConfig {
		if !taskConfig.Enabled {
			continue
		}
		var err error
		switch taskConfig.Task {
		case "chat_auto_resolve":
			_, err = scheduler.NewJob(
				gocron.CronJob(taskConfig.Schedule, false),
				gocron.NewTask(chatAutoResolve),
				gocron.WithSingletonMode(gocron.LimitModeReschedule),
			)
		case "chat_auto_open":
			_, err = scheduler.NewJob(
				gocron.CronJob(taskConfig.Schedule, false),
				gocron.NewTask(chatAutoOpen),
				gocron.WithSingletonMode(gocron.LimitModeReschedule),
			)
		case "service_ami_events":
			_, err = scheduler.NewJob(
				gocron.CronJob(taskConfig.Schedule, false),
				gocron.NewTask(serviceAmiEvents),
				gocron.WithSingletonMode(gocron.LimitModeReschedule),
			)
		default:
			utils.Logline("Unknown task", taskConfig.Task)
		}

		if err != nil {
			utils.Logline("Failed to schedule task", err)
		}
	}

	// Start the scheduler
	scheduler.Start()
}

// Define  task functions
func chatAutoResolve() {
	defer func() {
		if r := recover(); r != nil {
			utils.Logline("Recovered from panic <<chat_auto_resolve>>: %v", r)
		}
	}()

	//set variables for handling pgsql conn
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	db := models.ConnDb{Conn: PoolPgsql, Ctx: ctx}

	// run actual task
	if err := repo.ChatAutoResolve(db, "cronJob"); err != nil {
		utils.Logline("Error on chat_auto_resolve")
	}
}

func chatAutoOpen() {
	defer func() {
		if r := recover(); r != nil {
			utils.Logline("Recovered from panic <<chat_auto_open>>: %v", r)
		}
	}()

	//set variables for handling pgsql conn
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db := models.ConnDb{Conn: PoolPgsql, Ctx: ctx}

	// run actual task
	if err := repo.ChatAutoOpened(db, "cronJob"); err != nil {
		utils.Logline("Error on chat_auto_open")
	}
}

func serviceAmiEvents() {
	defer func() {
		if r := recover(); r != nil {
			utils.Logline("Recovered from panic <<service_ami_events>>: %v", r)
		}
	}()

	//set variables for handling mysql conn
	db := models.ConnMysql{Conn: PoolMysql, Ctx: context.Background()}

	// run actual task
	if err := repo.AmiEvents(db, "cronJob"); err != nil {
		utils.Logline("Error on service_ami_events")
	}
}
