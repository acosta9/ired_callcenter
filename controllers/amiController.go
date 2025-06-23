package controllers

import (
	"net/http"
	"strings"

	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
	"ired.com/callcenter/middlewares"
	"ired.com/callcenter/models"
	"ired.com/callcenter/repo"
)

func AmiRoutes(r *gin.Engine) {
	cron := r.Group("/ami")
	{
		cron.POST("/hangup-call", middlewares.ApiRestAuth(), hangupCall)
	}
}

// @Summary 			Colgar llamada de una extension
// @Description 	recibe un numero de extension y cuelga la llamada que tenga la extension abierta
// @Tags 					Ami
// @Accept 				json
// @Produce 			json
// @Security 			BasicAuth
// @Param 				user body models.ExtensionReq true "Extension Data"
// @Success 200 	{object} models.SuccessResponse
// @Failure 400 	{object} models.ErrorResponse
// @Router 				/ami/hangup-call [post]
func hangupCall(c *gin.Context) {
	// validate if body exist
	if c.Request.ContentLength == 0 {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: ginI18n.MustGetMessage(c, "errorFailedBody")},
		)
		return
	}

	// Bind and Validate the data and the struct
	var extensionReq models.ExtensionReq
	if err := c.ShouldBindJSON(&extensionReq); err != nil {
		if strings.Contains(err.Error(), "invalid character") || strings.Contains(err.Error(), "unmarshal") {
			c.AbortWithStatusJSON(
				http.StatusBadRequest,
				models.ErrorResponse{Error: ginI18n.MustGetMessage(c, "invalidJson")},
			)
			return
		}

		errorFormJson := models.ParseError(err, c)
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: errorFormJson},
		)
		return
	}

	// save user and check for errors
	err := repo.HangupCall(extensionReq.Extension)

	// validar por errores
	if err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			models.ErrorResponse{Error: err.Error()},
		)
		return
	}

	c.JSON(
		http.StatusOK,
		models.SuccessResponse{Notice: ginI18n.MustGetMessage(c, "formOK")},
	)
}
