package token

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"mingin/config"
	"mingin/model"
	"mingin/pkg/errno"
	"mingin/pkg/logger"
	"mingin/utils"
	"time"
)

var (
	// ErrMissingHeader means the `Authorization` header was empty.
	ErrMissingHeader = errors.New("the length of the `Authorization` header is zero")
)

type Claims struct {
	URole model.UserType `json:"u_role"` // 用户角色
	UID   uint64         `json:"uid"`    // 用户 ID
	UName string         `json:"u_name"` // 用户 nickname
	UAddr string         `json:"u_addr"` // 用户 addr
	RtID  uint64         `json:"rt_id"`  // refresh token id，若为 refresh token 该值为0
	jwt.StandardClaims
}

func SignWithClaims(c Claims, sc jwt.StandardClaims, s string) (TokenS string, err error) {
	if (jwt.StandardClaims{} != sc) {
		c.StandardClaims = sc
	}
	Token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	TokenS, err = Token.SignedString([]byte(s))
	if err != nil {
		return "", err
	}
	TokenS = base64.URLEncoding.EncodeToString([]byte(TokenS))
	return
}

func SignAccessToken(c Claims, rID uint64) (accessTokenString string, err error) {
	startTime := time.Now()
	atExpiresAt := time.Now().Add(time.Minute * 15)
	c.RtID = rID

	accessTokenString, err = SignWithClaims(c, jwt.StandardClaims{
		ExpiresAt: atExpiresAt.Unix(),
		IssuedAt:  startTime.Unix(),
		NotBefore: startTime.Unix(),
	}, config.GetMineGinJwtAccessSecret())
	if err != nil {
		return "", err
	}

	//access := model.UserATokenModel{
	//	AccessToken: accessTokenString,
	//	UserUID:     c.UID,
	//	ExpiresAt:   atExpiresAt,
	//	IssuedAt:    startTime,
	//	NotBefore:   startTime,
	//}
	//
	//if err = access.RegisterAccessToken(); err != nil {
	//	return "", err
	//}

	return
}

func SignRefreshToken(c Claims) (refreshTokenString string, refreshID uint64, err error) {
	startTime := time.Now()
	rtExpiresAt := time.Now().Add(time.Hour * 24 * 7)
	c.RtID = 0

	refreshTokenString, err = SignWithClaims(c, jwt.StandardClaims{
		ExpiresAt: rtExpiresAt.Unix(),
		IssuedAt:  startTime.Unix(),
		NotBefore: startTime.Unix(),
	}, config.GetMineGinJwtRefreshSecret())

	if err != nil {
		return "", 0, err
	}

	refresh := model.UserRTokenModel{
		RefreshToken: refreshTokenString,
		UserUID:      c.UID,
		UserAddr:     c.UAddr,
		UserName:     c.UName,
		ExpiresAt:    rtExpiresAt,
		IssuedAt:     startTime,
		NotBefore:    startTime,
	}

	if refreshID, err = refresh.RegisterRefreshToken(); err != nil {
		return "", 0, err
	}

	logger.InfoF("register token for client [%s] ", c.UAddr)

	return
}

func Sign(c Claims) (accessTokenString string, refreshTokenString string, err error) {
	var rID uint64
	refreshTokenString, rID, err = SignRefreshToken(c)
	if err != nil {
		return "", "", err
	}

	accessTokenString, err = SignAccessToken(c, rID)
	if err != nil {
		return "", "", err
	}

	return
}

//func SignWithRefresh(c *gin.Context) (*Claims, error) {
//
//}

// secretFunc 验证密钥格式
func secretFunc(secret string) jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	}
}

// Parse 使用指定的 secret 验证 token ，有效则返回 token 内容。
func Parse(tokenString string, secret string) (*Claims, error) {
	ctx := &Claims{}

	token, err := jwt.Parse(tokenString, secretFunc(secret))

	if err != nil {
		logger.ErrorF("Parse Token error with err: %s", err.Error())
		return ctx, errno.ErrTokenInvalid
	} else if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		ctx.UID = uint64(claims["uid"].(float64))
		ctx.URole = model.UserType(claims["u_role"].(float64))
		ctx.UName = claims["u_name"].(string)
		ctx.UAddr = claims["u_addr"].(string)
		ctx.RtID = uint64(claims["rt_id"].(float64))

		return ctx, nil
	} else {
		return ctx, err
	}
}

// ParseRequest 从 HTTP 请求头获取 token
// 并将其传递给 Parse 函数以验证 token 有消息。
func ParseRequest(c *gin.Context) (*Claims, error) {
	header := c.Request.Header.Get("Authorization")
	secret := config.GetMineGinJwtAccessSecret()

	if len(header) == 0 {
		return &Claims{}, ErrMissingHeader
	}

	var t string
	fmt.Sscanf(header, "Bearer %s", &t)
	bt, _ := base64.URLEncoding.DecodeString(t)

	claims, err := Parse(string(bt), secret)
	if err != nil {
		return nil, err
	}

	// 获取 url 中的 id 参数，不存在时 err 非空，此时无需处理。
	if uid, err := utils.GetUint64ByContext(c, "id"); err == nil {
		if uid != claims.UID {
			return nil, errno.ErrUID
		}
	}

	return claims, err
}

// ParseRefreshTokenRequest 从 HTTP 请求头获取 refresh token
// 并将其传递给 Parse 函数以验证 token 合法性。
func ParseRefreshTokenRequest(c *gin.Context) (*Claims, error) {
	header := c.Request.Header.Get("Authorization")
	secret := config.GetMineGinJwtRefreshSecret()

	if len(header) == 0 {
		return &Claims{}, ErrMissingHeader
	}

	var t string
	fmt.Sscanf(header, "Bearer %s", &t)
	bt, _ := base64.URLEncoding.DecodeString(t)

	claims, err := Parse(string(bt), secret)
	if err != nil {
		return nil, err
	}

	if claims.UAddr != utils.GetAddrFromContext(c) {
		logger.ErrorF("error token from %s (%v)",
			utils.GetAddrFromContext(c), claims.UName)
		return nil, errno.ErrClient
	}

	if _, err = model.ReTokenEffective(t); err != nil {
		return nil, errno.ErrRTokenInvalid
	}

	// Refresh Token 仅能验证一次，验证成功颁发一组新 Token，无论成功与否该 rt 必须失效
	if err = model.LogoutRefreshTokenWithToken(t); err != nil {
		return nil, err
	}

	return claims, nil
}
