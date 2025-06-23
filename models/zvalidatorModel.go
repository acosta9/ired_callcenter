package models

import (
	"errors"
	"regexp"
	"strings"

	ginI18n "github.com/gin-contrib/i18n"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type errorMsgs struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func errorToJson(fieldError validator.FieldError, c *gin.Context) string {
	switch fieldError.Tag() {
	case "required":
		return ginI18n.MustGetMessage(c, "veRequired")
	case "number":
		return ginI18n.MustGetMessage(c, "veNumber")
	case "numeric":
		return ginI18n.MustGetMessage(c, "veNumeric")
	case "min":
		return ginI18n.MustGetMessage(c, "veMinChar") + " " + fieldError.Param() + " " + ginI18n.MustGetMessage(c, "veChar")
	case "max":
		return ginI18n.MustGetMessage(c, "veMaxChar") + " " + fieldError.Param() + " " + ginI18n.MustGetMessage(c, "veChar")
	case "gte":
		return ginI18n.MustGetMessage(c, "veGte") + " " + fieldError.Param()
	case "lte":
		return ginI18n.MustGetMessage(c, "veLte") + " " + fieldError.Param()
	case "notzero":
		return ginI18n.MustGetMessage(c, "veNotzero") + " " + fieldError.Param()
	case "alfanumspa":
		return ginI18n.MustGetMessage(c, "veAlphaNumSpa")
	case "gte_number":
		return ginI18n.MustGetMessage(c, "veGte") + " " + fieldError.Param()
	case "lte_number":
		return ginI18n.MustGetMessage(c, "veLte") + " " + fieldError.Param()
	}
	return fieldError.Error() // default error
}

func ParseError(err error, c *gin.Context) []errorMsgs {
	var validatorError validator.ValidationErrors
	if errors.As(err, &validatorError) {
		out := make([]errorMsgs, len(validatorError))
		for i, fieldError := range validatorError {
			out[i] = errorMsgs{strings.ToLower(fieldError.Field()), errorToJson(fieldError, c)}
		}
		return out
	}
	return nil
}

var alphaNumEs validator.Func = func(fl validator.FieldLevel) bool {
	hasWhitespace := strings.TrimSpace(fl.Field().String()) != fl.Field().String()
	if hasWhitespace {
		return false
	}
	regex := regexp.MustCompile(`^[a-z A-Z0-9ñÑáéíóúÁÉÍÓÚ]+$`)
	return regex.MatchString(fl.Field().String())
}
