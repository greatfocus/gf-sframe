package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	gfjwt "github.com/greatfocus/gf-jwt"
)

// Token struct
type Token struct {
	Role        string
	Permissions []string
	UserID      int64
}

// JWT struct
type JWT struct {
	Secret     string
	Authorized bool
	Minutes    int64
	algorithm  gfjwt.Algorithm
}

// Init method prepare module
func (j *JWT) Init() {
	jwtMinutes, err := strconv.ParseUint(os.Getenv("JWT_Minutes"), 0, 64)
	if err != nil {
		log.Fatal(fmt.Println(err))
	}

	jwtAuthorized, err := strconv.ParseBool(os.Getenv("JWT_Authorized"))
	if err != nil {
		log.Fatal(fmt.Println(err))
	}

	j.Secret = os.Getenv("JWT_Secret")
	j.Authorized = jwtAuthorized
	j.Minutes = int64(jwtMinutes)
	j.algorithm = gfjwt.HmacSha256(j.Secret)
}

// CreateToken generates jwt for API login
func (j *JWT) CreateToken(userID int64, role string, permissions []string) (string, error) {
	claims := gfjwt.NewClaim()
	claims.Set("authorized", j.Authorized)
	claims.Set("userID", userID)
	claims.Set("role", role)
	claims.Set("permissions", permissions)
	claims.Set("exp", time.Now().Add(time.Minute*time.Duration(j.Minutes)).Unix()) //JWT expires after 1 hour
	token, err := j.algorithm.Encode(claims)
	if err != nil {
		return "", err
	}

	return token, nil
}

// TokenValid checks for jwt validity
func (j *JWT) TokenValid(r *http.Request) error {
	token := j.extractToken(r)
	err := j.algorithm.Validate(token)
	if err != nil {
		return err
	}
	return nil
}

// extractToken get jwt from header
func (j *JWT) extractToken(r *http.Request) string {
	keys := r.URL.Query()
	jwt := keys.Get("jwt")
	if jwt != "" {
		return jwt
	}
	bearerToken := r.Header.Get("Authorization")
	if len(strings.Split(bearerToken, " ")) == 2 {
		return strings.Split(bearerToken, " ")[1]
	}
	return ""
}

// GetToken get jwt id from header
func (j *JWT) GetToken(r *http.Request) (Token, error) {
	tokenString := j.extractToken(r)
	claims, err := j.algorithm.Decode(tokenString)
	if err != nil {
		return Token{}, err
	}

	userID, _ := claims.Get("userID")
	role, _ := claims.Get("Role")
	permissions, _ := claims.Get("permissions")
	var token = Token{
		UserID:      userID.(int64),
		Role:        role.(string),
		Permissions: permissions.([]string),
	}

	return token, nil
}
