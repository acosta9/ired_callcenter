package repo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/staskobzar/goami2"
	"ired.com/callcenter/models"
	"ired.com/callcenter/utils"
)

func ExtensionStatus(db models.ConnMysql) ([]models.ExtensionStatus, error) {
	queueMembers, err := getAmiQueueStatus()
	if err != nil {
		utils.Logline("error getting queue status", err)
	}

	var extensions []models.ExtensionStatus

	// Get extensions registered on call_center
	rowsMysql, err := db.Conn.QueryContext(db.Ctx, `SELECT number FROM call_center.agent WHERE agent.estatus='A' ORDER BY number ASC`)
	if err != nil {
		utils.Logline("error on getting users from mysql", err)
		return nil, err
	}
	defer rowsMysql.Close()

	for rowsMysql.Next() {
		var extension, status string
		if err := rowsMysql.Scan(&extension); err != nil {
			utils.Logline("error passing the usersId to an array: ", err)
			return nil, err
		}

		if status, err = getAmiExtStatus(extension); err != nil {
			utils.Logline("error getting status of extension via ami", extension, err)
			status = "-"
		}

		onQueue := checkExtenOnQueue(extension, queueMembers)

		extensions = append(extensions, models.ExtensionStatus{Extension: extension, Status: status, OnQueue: onQueue})
	}
	rowsMysql.Close()

	return extensions, nil
}

func getAmiExtStatus(exten string) (string, error) {
	// Connect to Asterisk AMI
	clientAmi, err := utils.ConnectToAmi()
	if err != nil {
		return "", err
	}
	defer clientAmi.Close()

	// Retrieve extension status
	action := goami2.NewAction("ExtensionState")
	action.SetField("Exten", exten) // Replace with your extension number
	actionID := fmt.Sprintf("extstatus-%d", time.Now().Unix())
	action.SetField("ActionID", actionID)

	// Send the action
	clientAmi.Send(action.Byte())

	responseChan := make(chan string)
	errorChan := make(chan error)

	// create context
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	go func() {
		for {
			select {
			case <-ctx.Done():
				errorChan <- fmt.Errorf("timeout waiting for response")
				return
			case msg := <-clientAmi.AllMessages():
				if msg.ActionID() == actionID {
					status := msg.Field("Status")
					if status != "" {
						responseChan <- status
						return
					}
				}
			case err := <-clientAmi.Err():
				errorChan <- fmt.Errorf("ami error: %v", err)
				return
			}
		}
	}()

	// Wait for response or timeout
	select {
	case status := <-responseChan:
		return translateStatusExtension(status), nil
	case err := <-errorChan:
		return "", err
	case <-ctx.Done():
		return "", fmt.Errorf("timeout waiting for extension status")
	}
}

func getAmiQueueStatus() ([]models.QueueMember, error) {
	// Connect to Asterisk AMI
	clientAmi, err := utils.ConnectToAmi()
	if err != nil {
		return nil, err
	}
	defer clientAmi.Close()

	// Retrieve extension status
	action := goami2.NewAction("QueueStatus")
	action.SetField("Qeueue", "8000") // Replace with your extension number
	actionID := fmt.Sprintf("queuestatus-%d", time.Now().Unix())
	action.SetField("ActionID", actionID)

	// Send the action
	clientAmi.Send(action.Byte())

	// create context
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// var queueMembers []models.QueueMember
	var queueMembers []models.QueueMember

	// Listen for responses
	for {
		select {
		case msg := <-clientAmi.AllMessages():
			if msg.Field("Event") == "QueueMember" {
				queueName := msg.Field("queue")
				exten := strings.ReplaceAll(msg.Field("name"), "SIP/", "")
				status := msg.Field("status")
				queueMembers = append(queueMembers, models.QueueMember{QueueName: queueName, Extension: exten, Status: status})
			}

			// Break the loop if the response is "QueueStatusComplete"
			if msg.Field("Event") == "QueueStatusComplete" {
				fmt.Println("QueueStatus completed.")
				return queueMembers, nil
			}

		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for extension status")

		case err := <-clientAmi.Err():
			return nil, err
		}
	}
}

func checkExtenOnQueue(exten string, queue []models.QueueMember) bool {
	for _, member := range queue {
		if exten == member.Extension {
			return true
		}
	}
	return false
}

// translateStatusExtension converts numeric status to human-readable format
func translateStatusExtension(status string) string {
	switch status {
	case "0":
		return "Idle"
	case "1":
		return "InUse"
	case "2":
		return "Busy"
	case "4":
		return "Unavailable"
	case "8":
		return "Ringing"
	case "16":
		return "OnHold"
	default:
		return "Unknown"
	}
}
