package controllers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/qicfan/backup-server/helpers"
)

type APIResponseCode int

const (
	Success APIResponseCode = iota
	BadRequest
	TerminalConnection = 3 // 用于WebSocket连接断开
	ContinueTransfer   = 4 // 用于WebSocket继续传输
)

type APIResponse[T any] struct {
	Code    APIResponseCode `json:"code"`
	Message string          `json:"message"`
	Data    T               `json:"data"`
}

type LoginUser struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

var jwtSecret = []byte("backup-server-2508270000")

// JWTAuthMiddleware 基于JWT的认证中间件--验证用户是否登录
func JWTAuthMiddleware() func(c *gin.Context) {
	return func(c *gin.Context) {
		authHeader := c.Request.Header.Get("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, APIResponse[any]{Code: BadRequest, Message: "Token不存在", Data: nil})
			c.Abort()
			return
		}
		// 按空格分割
		parts := strings.Split(authHeader, ".")
		if len(parts) != 3 {
			c.JSON(http.StatusUnauthorized, APIResponse[any]{Code: BadRequest, Message: "Token格式有误", Data: nil})
			c.Abort()
			return
		}
		tokenString := strings.Replace(authHeader, "Bearer ", "", 1)
		loginUser, err := ValidateJWT(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, APIResponse[any]{Code: BadRequest, Message: fmt.Sprintf("Token无效：%v", err), Data: nil})
			c.Abort()
			return
		}
		helpers.AppLogger.Infof("Authenticated user: %s", loginUser.Username)
		// 将当前请求的username信息保存到请求的上下文c上
		c.Set("username", loginUser.Username)
		c.Next() // 后续的处理函数可以用过c.Get("username")来获取当前请求的用户信息
	}
}

func ValidateJWT(tokenString string) (*LoginUser, error) {
	token, err := jwt.ParseWithClaims(tokenString, &LoginUser{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("登录凭证校验失败: %v", err)
	}
	claims := token.Claims.(*LoginUser)
	if claims.Username == "" {
		return nil, fmt.Errorf("登录凭证中无法获取用户名")
	}

	return claims, nil
}
