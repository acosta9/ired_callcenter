package repo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"ired.com/callcenter/models"
	"ired.com/callcenter/utils"
)

type convToOpen struct {
	Id          int
	DisplayId   int
	ContactId   int
	ContactName string
}

type newMsg struct {
	Content     string `json:"content"`
	MessageType string `json:"message_type"`
	Private     bool   `json:"private"`
}

type newMsgResponse struct {
	Error             string      `json:"error,omitempty"`
	ID                int         `json:"id"`
	Content           string      `json:"content"`
	InboxID           int         `json:"inbox_id"`
	ConversationID    int         `json:"conversation_id"`
	MessageType       int         `json:"message_type"`
	ContentType       string      `json:"content_type"`
	Status            string      `json:"status"`
	ContentAttributes struct{}    `json:"content_attributes"`
	CreatedAt         int         `json:"created_at"`
	Private           bool        `json:"private"`
	SourceID          interface{} `json:"source_id"`
	Sender            struct {
		ID                 int    `json:"id"`
		Name               string `json:"name"`
		AvailableName      string `json:"available_name"`
		AvatarURL          string `json:"avatar_url"`
		Type               string `json:"type"`
		AvailabilityStatus string `json:"availability_status"`
		Thumbnail          string `json:"thumbnail"`
	} `json:"sender"`
}

type toggleData struct {
	Status string `json:"status"`
}

type toogleResponse struct {
	Meta    struct{} `json:"meta"`
	Error   string   `json:"error,omitempty"`
	Errors  []string `json:"errors,omitempty"`
	Payload struct {
		Success        bool   `json:"success"`
		CurrentStatus  string `json:"current_status"`
		ConversationID int    `json:"conversation_id"`
	} `json:"payload"`
}

func ChatAutoResolve(db models.ConnDb, caller string) error {
	//show status of worker
	utils.ShowStatusWorker(db, "chatAutoOpened", caller+"/begin")

	//lookUp for conversations that are mark pending and the latest message send was by an agent
	query := `WITH latest_msgs AS (
		SELECT q0.conversation_id, q0.message_type
			FROM (
				SELECT m.conversation_id, m.message_type, ROW_NUMBER() OVER (PARTITION BY m.conversation_id ORDER BY m.created_at DESC) AS rn
				FROM messages as m
				LEFT JOIN conversations as c ON c.id=m.conversation_id
				WHERE m.message_type IN (0, 1) AND m.created_at >= DATE(NOW() - INTERVAL '3 month')::timestamp AND c.status=2
			) as q0
			WHERE q0.rn=1
		), conversations_pending AS (
			SELECT q0.conversation_id, q0.display_id, ROUND(q0.date_diff/60) as min_ago, q0.contact_id
			FROM (
				SELECT c.id as conversation_id, c.display_id, EXTRACT(EPOCH FROM NOW() - MAX(m.created_at - INTERVAL '4 hours')) as date_diff, c.contact_id
				FROM conversations AS c
				LEFT JOIN messages AS m ON m.conversation_id = c.id
				WHERE c.status = 2 AND m.message_type IN (0, 1) AND m.created_at>=NOW()-INTERVAL'3 month'
				GROUP BY c.id
			) as q0
			WHERE (q0.date_diff/60)>720
		)

		SELECT cp.conversation_id, cp.display_id, LOWER(contacts.name) as contact_name, cp.contact_id
		FROM conversations_pending as cp
		LEFT JOIN latest_msgs as lm ON lm.conversation_id=cp.conversation_id
		LEFT JOIN contacts ON contacts.id=cp.contact_id
		WHERE lm.message_type=1
		ORDER BY cp.min_ago DESC
		LIMIT 100
	`

	rows, err := db.Conn.Query(db.Ctx, query)
	if err != nil {
		utils.Logline("error getting conversations pending", "chatAutoResolve", err)
		return err
	}
	defer rows.Close()

	var conversations []convToOpen
	for rows.Next() {
		var conv convToOpen
		if err := rows.Scan(&conv.Id, &conv.DisplayId, &conv.ContactName, &conv.ContactId); err != nil {
			utils.Logline("error scanning conversationsId pending", "chatAutoResolve", err)
			return err
		}
		conversations = append(conversations, conv)
	}

	for _, conv := range conversations {
		if err := sendMsg(conv.Id, conv.DisplayId, "Se cambia estatus a resuelto, sin respuesta del cliente pasadas 12h"); err != nil {
			utils.Logline(fmt.Sprintf("Error creating new msg conv_id (%d), display_id (%d), contacto(%d : %s)", conv.Id, conv.DisplayId, conv.ContactId, conv.ContactName), "chatAutoResolve", err)
			continue
		}
		if err := toogleStatus(conv.Id, conv.DisplayId, "resolved"); err != nil {
			utils.Logline(fmt.Sprintf("Error change conv to resolved, conv_id (%d), display_id (%d), contacto(%d : %s)", conv.Id, conv.DisplayId, conv.ContactId, conv.ContactName), "chatAutoResolve", err)
			continue
		}
		utils.Logline(fmt.Sprintf("Success change conv to resolved, conv_id (%d), display_id (%d), contacto(%d : %s)", conv.Id, conv.DisplayId, conv.ContactId, conv.ContactName), "chatAutoResolve")
	}

	//show status of worker
	utils.ShowStatusWorker(db, "chatAutoResolve", caller+"/ending")

	return nil
}

func ChatAutoOpened(db models.ConnDb, caller string) error {
	//show status of worker
	utils.ShowStatusWorker(db, "chatAutoOpened", caller+"/begin")

	//lookUp for conversations that are mark pending and the latest message recieve was from a client
	query := `WITH latest_msgs AS (
			SELECT q0.conversation_id, q0.message_type
			FROM (
				SELECT m.conversation_id, m.message_type, ROW_NUMBER() OVER (PARTITION BY m.conversation_id ORDER BY m.created_at DESC) AS rn
				FROM messages as m
				LEFT JOIN conversations as c ON c.id=m.conversation_id
				WHERE m.message_type IN (0, 1) AND m.created_at >= DATE(NOW() - INTERVAL '3 month')::timestamp AND c.status=2
			) as q0
			WHERE q0.rn=1
		)

		SELECT c.id, c.display_id, LOWER(contacts.name) as contacto, contacts.id as contact_id
		FROM conversations AS c
		LEFT JOIN latest_msgs AS lm ON lm.conversation_id=c.id
		LEFT JOIN contacts ON contacts.id=c.contact_id
		WHERE lm.message_type = 0 AND c.status=2
		ORDER BY c.id ASC`

	rows, err := db.Conn.Query(db.Ctx, query)
	if err != nil {
		utils.Logline("error getting conversations pending", "chatAutoOpened", err)
		return err
	}
	defer rows.Close()

	var conversations []convToOpen
	for rows.Next() {
		var conv convToOpen
		if err := rows.Scan(&conv.Id, &conv.DisplayId, &conv.ContactName, &conv.ContactId); err != nil {
			utils.Logline("error scanning conversationsId pending", "chatAutoOpened", err)
			return err
		}
		conversations = append(conversations, conv)
	}

	for _, conv := range conversations {
		if err := toogleStatus(conv.Id, conv.DisplayId, "open"); err != nil {
			utils.Logline(fmt.Sprintf("Error change conv status to open conv_id(%d), display_id(%d), contacto(%d : %s)", conv.Id, conv.DisplayId, conv.ContactId, conv.ContactName), "chatAutoOpened", err)
			continue
		}
		utils.Logline(fmt.Sprintf("Success change conv status to open conv_id(%d), display_id(%d), contacto(%d : %s)", conv.Id, conv.DisplayId, conv.ContactId, conv.ContactName), "chatAutoOpened")
	}

	//show status of worker
	utils.ShowStatusWorker(db, "chatAutoOpened", caller+"/ending")

	return nil
}

func toogleStatus(convId int, displayId int, status string) error {
	apiToken := os.Getenv("CHAT_TOKEN")
	toggleURL := os.Getenv("CHAT_TOGGLE_URL")

	toggleData := toggleData{Status: status}
	togglePayload, _ := json.Marshal(toggleData)

	// Create a new request
	req, err := http.NewRequest("POST", fmt.Sprintf(toggleURL, displayId), bytes.NewBuffer(togglePayload))
	if err != nil {
		return err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api_access_token", apiToken) // Add the custom header

	session := &http.Client{}

	// Send the request
	toggleResp, err := session.Do(req)
	if err != nil {
		return err
	}
	defer toggleResp.Body.Close()

	body, _ := io.ReadAll(toggleResp.Body)

	// Convert JSON string to slice
	var response toogleResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return err
	}

	if len(response.Error) > 2 {
		return fmt.Errorf("%s", response.Error)
	}

	if len(response.Errors) >= 1 {
		return fmt.Errorf("%s", strings.Join(response.Errors, ", "))
	}

	if os.Getenv("GIN_MODE") == "debug" {
		utils.Logline("response from api", "toogleStatus", convId, displayId, response)
	}

	return nil
}

func sendMsg(convId int, displayId int, content string) error {
	apiToken := os.Getenv("CHAT_TOKEN")
	newMsgURL := os.Getenv("CHAT_NEWMSG_URL")

	bodyInfo := newMsg{
		Content:     content,
		MessageType: "outgoing",
		Private:     true,
	}
	bodyPayload, _ := json.Marshal(bodyInfo)

	// Create a new request
	req, err := http.NewRequest("POST", fmt.Sprintf(newMsgURL, displayId), bytes.NewBuffer(bodyPayload))
	if err != nil {
		return err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api_access_token", apiToken) // Add the custom header

	session := &http.Client{}

	// Send the request
	toggleResp, err := session.Do(req)
	if err != nil {
		return err
	}
	defer toggleResp.Body.Close()

	body, _ := io.ReadAll(toggleResp.Body)

	// Convert JSON string to slice
	var response newMsgResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return err
	}

	if len(response.Error) >= 1 {
		return fmt.Errorf("%s", response.Error)
	}

	if os.Getenv("GIN_MODE") == "debug" {
		utils.Logline("response from api", "newMsg", convId, displayId, response)
	}

	return nil
}

// messages 		-> message_type
// 	0 cliente
// 	1 agente
// 	2 acciones App (marcada pendiente, reabierta, resuelta, etc)
// 	3 autoRespuesta

// conversation -> status
// 	0 por abrir
// 	1 marcada como resuelto
// 	2 marcada como pendiente
