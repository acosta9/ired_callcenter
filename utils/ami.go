package utils

import (
	"fmt"
	"net"
	"os"

	"github.com/staskobzar/goami2"
)

func ConnectToAmi() (*goami2.Client, error) {
	// Connect to Asterisk AMI
	connPbx, err := net.Dial("tcp", os.Getenv("AMI_SERVER"))
	if err != nil {
		Logline("Error connecting to Asterisk", err)
		return nil, fmt.Errorf("error connecting to asterisk")
	}

	// Login to AMI
	clientAmi, err := goami2.NewClient(connPbx, os.Getenv("AMI_USER"), os.Getenv("AMI_PASSWD"))
	if err != nil {
		Logline("Error logging into AMI", err)
		return nil, fmt.Errorf("error logging into ami")
	}

	return clientAmi, nil
}
