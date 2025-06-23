package repo

import (
	"fmt"
	"regexp"
	"time"

	"github.com/staskobzar/goami2"
	"golang.org/x/net/context"
	"ired.com/callcenter/models"
	"ired.com/callcenter/utils"
)

// Global variables
var trackList = make(map[string]bool)  // list of all calls
var activeList = make(map[string]bool) // list of active calls (atendidas)

func AmiEvents(db models.ConnMysql, caller string) error {
	// Connect to Asterisk AMI
	clientAmi, err := utils.ConnectToAmi()
	if err != nil {
		return err
	}
	defer clientAmi.Close()

	utils.Logline("Starting AMI events service")
	for {
		select {
		case msg := <-clientAmi.AllMessages():
			if msg != nil {
				handleEvent(db, msg)
			}
		case err := <-clientAmi.Err():
			utils.Logline("error on ami", err)
			return fmt.Errorf("an error occurred executing ami command")
		}
	}
}

// funcion principal para lectura de eventos
// newChannel llamadaNueva solo las que inician con la extension 80*
// Hangup Colgar llamada de cualquiera de las dos partes
// BridgeEnter evento cuando atienden llamada
// BridgeLeave evento cuando la llamada termina
func handleEvent(db models.ConnMysql, msg *goami2.Message) {
	uniqueId := msg.Field("Uniqueid")
	linkedId := msg.Field("Linkedid")
	context := msg.Field("Context")

	if msg.IsEvent() {
		switch msg.Field("Event") {
		case "Newchannel":
			if trackList[linkedId] || uniqueId != linkedId {
				return
			}
			if match, _ := regexp.MatchString("^80.*", msg.Field("CallerIDNum")); match {
				utils.Logline("new event [newchannel] ", msg)
				trackList[linkedId] = true
				insertCall(db, msg)
			}
		case "Hangup":
			if context != "tc-maint" && trackList[linkedId] && !activeList[linkedId] {
				if match, _ := regexp.MatchString("^80.*", msg.Field("CallerIDNum")); match {
					utils.Logline("new event [hangup] ", msg)
					if err := endCall(db, msg); err != nil {
						delete(trackList, linkedId)
					}
				}
			}
		case "BridgeEnter":
			if trackList[linkedId] && uniqueId != linkedId {
				utils.Logline("new event [bridgeenter] ", msg)
				if err := bridgeEnterCall(db, msg); err != nil {
					activeList[linkedId] = true
				}
			}
		case "BridgeLeave":
			if activeList[linkedId] && uniqueId != linkedId {
				utils.Logline("new event [bridgeleave] ", msg)
				if err := endCall(db, msg); err != nil {
					delete(activeList, linkedId)
					delete(trackList, linkedId)
				}
			}
		}
	}
}

func insertCall(db models.ConnMysql, msg *goami2.Message) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	uniqueIdDb := msg.Field("Linkedid")
	callerIdNum := msg.Field("CallerIDNum")
	channel := msg.Field("Channel")

	channelClient := "-"
	if msg.Field("Context") == "from-internal" {
		channelClient = msg.Field("Exten")
	}

	agentId := getAgentId(db, callerIdNum)
	if agentId == 0 {
		utils.Logline("Failed to insert call: agentID is 0", msg)
		return fmt.Errorf("failed to insert call")
	}

	query := `INSERT INTO calls (id_campaign, phone, status, uniqueid, fecha_llamada, retries, id_agent, datetime_entry_queue, duration_wait, dnc, datetime_originate, trunk, scheduled)
		VALUES (1, ?, 'Ringing', ?, NOW(), 0, ?, NOW(), 0, 0, NOW(), ?, 0)`
	_, err := db.Conn.QueryContext(ctx, query, callerIdNum, uniqueIdDb, agentId, channelClient)
	if err != nil {
		utils.Logline("Failed to insert call: ", msg, err)
		return fmt.Errorf("failed to insert call")
	}

	var callId string
	err = db.Conn.QueryRowContext(ctx, `SELECT id FROM calls WHERE uniqueid=?`, uniqueIdDb).Scan(&callId)
	if err != nil {
		utils.Logline("Failed to insert call: ", msg, err)
		return fmt.Errorf("failed to insert call")
	}

	query = `INSERT INTO current_calls (id_call, fecha_inicio, uniqueid, queue, agentnum, event, Channel, ChannelClient, hold)
		VALUES (?, NOW(), ?, '8000', ?, 'Dialing', ?, ?, 'N')`
	_, err = db.Conn.QueryContext(ctx, query, callId, uniqueIdDb, callerIdNum, channel, channelClient)
	if err != nil {
		utils.Logline("Failed to insert current_call: ", msg, err)
		return fmt.Errorf("failed to insert current_call")
	}

	return nil
}

func bridgeEnterCall(db models.ConnMysql, msg *goami2.Message) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	uniqueIdDb := msg.Field("Linkedid")

	var err error
	var callId, phone, callStatus string
	err = db.Conn.QueryRowContext(ctx, "SELECT id, phone, LOWER(status) FROM calls WHERE uniqueid=?", uniqueIdDb).Scan(&callId, &phone, &callStatus)
	if err != nil {
		utils.Logline("Failed to read status of call in db: ", msg, err)
		return fmt.Errorf("failed to read status of call in db")
	}

	var query string
	// el primer IF sirve para evaluar cuando la llamada fue finalizada por que la transfirieron
	if callStatus == "finalizada" || callStatus == "sin respuesta" {
		query = `UPDATE calls SET status='Active', transfer=? WHERE uniqueid = ?`
		_, err = db.Conn.QueryContext(ctx, query, msg.Field("CallerIDNum"), uniqueIdDb)
		if err != nil {
			utils.Logline("failed to update previous uniqueid call on db", msg, err)
			return fmt.Errorf("failed to update previous uniqueid call on db")
		}

		channel := msg.Field("Channel")
		channelClient := msg.Field("ConnectedLineNum")

		query = `INSERT INTO current_calls (id_call, fecha_inicio, uniqueid, queue, agentnum, event, Channel, ChannelClient, hold)
			VALUES (?, NOW(), ?, '8000', ?, 'Dialing', ?, ?, 'N')`
		_, err = db.Conn.QueryContext(ctx, query, callId, uniqueIdDb, phone, channel, channelClient)
		if err != nil {
			utils.Logline("Failed to insert current_call: ", msg, err)
			return fmt.Errorf("failed to insert current_call")
		}

		trackList[msg.Field("Linkedid")] = true
		activeList[msg.Field("Linkedid")] = true
	} else {
		query := `UPDATE calls SET status = 'Active', start_time = NOW(), duration_wait = TIMESTAMPDIFF(SECOND, fecha_llamada, NOW()) WHERE uniqueid = ?`
		_, err = db.Conn.QueryContext(ctx, query, uniqueIdDb)
		if err != nil {
			utils.Logline("Failed to start call", msg, err)
			return fmt.Errorf("failed to start call")
		}

		query = `UPDATE current_calls SET event = 'Link' WHERE uniqueid = ?`
		_, err = db.Conn.QueryContext(ctx, query, uniqueIdDb)
		if err != nil {
			utils.Logline("Failed to start current_call", msg, err)
			return fmt.Errorf("failed to start current_call")
		}
	}

	return nil
}

func endCall(db models.ConnMysql, msg *goami2.Message) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var query string
	var err error
	uniqueIdDb := msg.Field("Linkedid")

	var callStatus string
	err = db.Conn.QueryRowContext(ctx, "SELECT LOWER(status) FROM calls WHERE uniqueid=?", uniqueIdDb).Scan(&callStatus)
	if err != nil {
		utils.Logline("Failed to read status of call in db: ", msg, err)
		return fmt.Errorf("failed to read status of call in db")
	}

	// si la llamada fue colgada y ya habia sido atendida se categoriza como finalizada
	// si se cuelga cuando solo estaba ringing entonces no se optuvo respuesta de la contraparte
	if callStatus == "active" {
		query := `UPDATE calls SET status = 'Finalizada', end_time = NOW(), duration = TIMESTAMPDIFF(SECOND, start_time, NOW()) WHERE uniqueid = ? AND (end_time IS NULL OR transfer<>'') `
		_, err = db.Conn.QueryContext(ctx, query, uniqueIdDb)
		if err != nil {
			utils.Logline("Failed to update call status", msg, err)
			return fmt.Errorf("failed to update call status")
		}
	} else if callStatus == "ringing" {
		query = `UPDATE calls SET status = 'Sin respuesta', end_time = NOW(), duration_wait = TIMESTAMPDIFF(SECOND, fecha_llamada, NOW()), duration = TIMESTAMPDIFF(SECOND, fecha_llamada, NOW()) 
			WHERE uniqueid = ? AND (end_time IS NULL OR transfer<>'')`
		_, err = db.Conn.QueryContext(ctx, query, uniqueIdDb)
		if err != nil {
			utils.Logline("Failed to end call", msg, err)
			return fmt.Errorf("failed to end call")
		}
	}

	query = `DELETE FROM current_calls WHERE uniqueid = ?`
	_, err = db.Conn.QueryContext(ctx, query, uniqueIdDb)
	if err != nil {
		utils.Logline("Failed to end current_calls", msg, err)
		return fmt.Errorf("failed to end current_calls")
	}

	return nil
}

func getAgentId(db models.ConnMysql, callerId string) int {
	var agentId int

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	query := `SELECT id FROM agent WHERE number = ? AND estatus = 'A'`
	err := db.Conn.QueryRowContext(ctx, query, callerId).Scan(&agentId)
	if err != nil {
		utils.Logline("Failed to retrieve agent ID: %v", err)
		return 0
	}
	return agentId
}
