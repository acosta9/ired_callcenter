package controllers

import (
	"context"
	"net/http"
	"time"

	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
	"ired.com/callcenter/app"
	"ired.com/callcenter/middlewares"
	"ired.com/callcenter/models"
	"ired.com/callcenter/repo"
)

func GrafanaRoutes(r *gin.Engine) {
	cron := r.Group("/grafana")
	{
		cron.GET("/get-extension-status", middlewares.GrafanaAuth(), extensionStatus)
	}
}

// @Summary 			Get Extension Status from PBX
// @Description 	get the extension status of all extension of agents in call center using AMI connection to asterisk
// @Tags 					Grafana
// @Accept 				json
// @Produce 			json
// @Security 			BasicAuth
// @Success 			200 {object} models.SuccessResponse
// @Failure 			400 {object} models.ErrorResponse
// @Router 				/grafana/get-extension-status [get]
func extensionStatus(c *gin.Context) {
	//set variables for handling pgsql conn
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db := models.ConnMysql{Conn: app.PoolMysql, Ctx: ctx}

	extensions, err := repo.ExtensionStatus(db)
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: err.Error()},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		models.SuccessResponse{
			Notice: ginI18n.MustGetMessage(c, "queryOK"),
			Record: extensions,
		},
	)
}
