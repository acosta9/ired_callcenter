package models

type ExtensionReq struct {
	Extension string `json:"extension" binding:"required,number,min=4,max=5"`
}
