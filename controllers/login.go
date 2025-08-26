package controllers

import (
	"fmt"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte("your-secret-key")

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type LoginResponse struct {
	Token string `json:"token"`
}

func HandleLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, APIResponse[interface{}]{Code: BadRequest, Message: "参数错误", Data: nil})
		return
	}
	envUser := os.Getenv("USERNAME")
	if envUser == "" {
		envUser = "admin"
	}
	envPass := os.Getenv("PASSWORD")
	if envPass == "" {
		envPass = "admin"
	}
	if req.Username != envUser || req.Password != envPass {
		c.JSON(200, APIResponse[interface{}]{Code: BadRequest, Message: "用户名或密码错误", Data: nil})
		return
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": req.Username,
		"exp":      jwt.NewNumericDate(time.Now().Add(365 * 24 * time.Hour)),
	})
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		c.JSON(200, APIResponse[interface{}]{Code: BadRequest, Message: "Token生成失败", Data: nil})
		return
	}
	c.JSON(200, APIResponse[map[string]string]{Code: Success, Message: "", Data: map[string]string{"token": tokenString}})
}

func ValidateJWT(tokenString string) error {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return fmt.Errorf("invalid token")
	}
	return nil
}
