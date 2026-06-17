package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Body struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Body{
		Code: 0,
		Msg:  "success",
		Data: data,
	})
}

func Error(c *gin.Context, httpStatus int, msg string) {
	c.JSON(httpStatus, Body{
		Code: -1,
		Msg:  msg,
		Data: nil,
	})
}