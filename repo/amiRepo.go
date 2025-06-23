package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/staskobzar/goami2"
	"ired.com/callcenter/utils"
)

func HangupCall(ext string) error {
	// Connect to Asterisk AMI
	clientAmi, err := utils.ConnectToAmi()
	if err != nil {
		return err
	}
	defer clientAmi.Close()

	// hangup-call
	action := goami2.NewAction("Hangup")
	action.SetField("Channel", fmt.Sprintf("/^SIP/%s-.*$/", ext)) // Replace with your extension number
	actionID := fmt.Sprintf("hangupcall-%d", time.Now().Unix())
	action.SetField("ActionID", actionID)

	// Send the action
	clientAmi.Send(action.Byte())

	// create context
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Listen for responses
	for {
		select {
		case msg := <-clientAmi.AllMessages():
			if msg.Field("Event") == "Hangup" {
				utils.Logline("extension colgada con exito", ext)
				return nil
			}

			// Break the loop if no channel its found
			if msg.Field("Message") == "No such channel" {
				return fmt.Errorf("no existe un canal abierto en la extension %s", ext)
			}

		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for hangup call")

		case err := <-clientAmi.Err():
			utils.Logline("error on ami", err)
			return fmt.Errorf("an error occurred executing ami command")
		}
	}
}
