package models

type ExtensionStatus struct {
	Extension string `json:"extension"`
	Status    string `json:"status"`
	OnQueue   bool   `json:"on_queue"`
}

type QueueMember struct {
	QueueName string `json:"queue_name"`
	Extension string `json:"extension"`
	Status    string `json:"status"`
}
