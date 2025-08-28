package controllers

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/qicfan/backup-server/helpers"
)

type LoginRequest struct {
	Username string `json:"username" form:"username"`
	Password string `json:"password" form:"password"`
}
type LoginResponse struct {
	Token string `json:"token"`
}

func HandleLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse[any]{Code: BadRequest, Message: fmt.Sprintf("参数错误：%v", err), Data: nil})
		return
	}
	helpers.AppLogger.Infof("Login attempt: %s, %s", req.Username, req.Password)
	envUser := os.Getenv("USERNAME")
	if envUser == "" {
		envUser = "admin"
	}
	envPass := os.Getenv("PASSWORD")
	if envPass == "" {
		envPass = "admin"
	}
	helpers.AppLogger.Infof("ENV attempt: %s, %s", envUser, envPass)
	if req.Username != envUser || req.Password != envPass {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "用户名或密码错误", Data: nil})
		return
	}
	claims := &LoginUser{
		ID:       1,
		Username: req.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(365 * 24 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		c.JSON(http.StatusOK, APIResponse[any]{Code: BadRequest, Message: "Token生成失败", Data: nil})
		return
	}
	c.JSON(http.StatusOK, APIResponse[map[string]string]{Code: Success, Message: "", Data: map[string]string{"token": tokenString}})
}
