package platform

import "github.com/gin-gonic/gin"

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Response struct {
	Data  any        `json:"data"`
	Error *ErrorBody `json:"error"`
}

func Respond(c *gin.Context, status int, data any, apiError *ErrorBody) {
	c.JSON(status, Response{Data: data, Error: apiError})
}
