package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"ired.com/callcenter/app"
	"ired.com/callcenter/middlewares"
	"ired.com/callcenter/models"
	"ired.com/callcenter/repo"
)

func CronRoutes(r *gin.Engine) {
	cron := r.Group("/cron")
	{
		cron.GET("/chat-auto-resolve", middlewares.BasicAuth(), chatAutoResolve)
		cron.GET("/chat-auto-opened", middlewares.BasicAuth(), chatAutoOpened)
	}
}

// @Summary 			Run the task chat_auto_resolve
// @Description 	run cron to autoResolve chats
// @Tags 					Crons
// @Accept 				json
// @Produce 			json
// @Security 			BasicAuth
// @Success 			200 {object} models.SuccessResponse
// @Failure 			400 {object} models.ErrorResponse
// @Router 				/cron/chat-autoresolve [get]
func chatAutoResolve(c *gin.Context) {
	//set variables for handling pgsql conn
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db := models.ConnDb{Conn: app.PoolPgsql, Ctx: ctx}

	if err := repo.ChatAutoResolve(db, "restApi"); err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: err.Error()},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		models.SuccessResponse{Notice: "Cron Executed ok"},
	)
}

// @Summary 			Run the task chat_auto_opened
// @Description 	run cron para cambiar estatus a por abrir de aquellos chats que fueron marcados como pendientes por un agente y el ultimo mensaje recibido fue del cliente
// @Tags 					Crons
// @Accept 				json
// @Produce 			json
// @Security 			BasicAuth
// @Success 			200 {object} models.SuccessResponse
// @Failure 			400 {object} models.ErrorResponse
// @Router 				/cron/chat-auto-opened [get]
func chatAutoOpened(c *gin.Context) {
	//set variables for handling pgsql conn
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db := models.ConnDb{Conn: app.PoolPgsql, Ctx: ctx}

	if err := repo.ChatAutoOpened(db, "restApi"); err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: err.Error()},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		models.SuccessResponse{Notice: "Cron Executed ok"},
	)
}
