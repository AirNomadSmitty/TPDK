package utils

import (
	"errors"
	"time"

	"github.com/dgrijalva/jwt-go"
)

var jwtKey = []byte("testKeyHere")

type Auth struct {
	UserID int64
}

type AuthClaims struct {
	UserID int64
	jwt.StandardClaims
}

func (auth *Auth) IsLoggedIn() bool {
	if auth.UserID != 0 {
		return true
	}
	return false
}

func (auth *Auth) MakeJWT() (*string, *time.Time, error) {
	expirationTime := time.Now().Add(time.Hour * 24 * 30)
	claims := &AuthClaims{
		UserID: auth.UserID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return nil, nil, err
	}

	return &tokenString, &expirationTime, nil
}

func GetAuthFromJWT(tokenString string) (*Auth, error) {
	claims := &AuthClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("Invalid token")
	}

	return &Auth{claims.UserID}, nil
}
