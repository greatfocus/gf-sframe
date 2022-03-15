package server

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	gfjwt "github.com/greatfocus/gf-jwt"
	"github.com/greatfocus/gf-sframe/model"
)

// Token struct
type Token struct {
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
func (j *JWT) CreateToken(userID int64, permissions []string, origin string) (*model.Bearer, error) {
	var err error
	td := &model.Bearer{}
	td.AtExpires = time.Now().Add(time.Minute * time.Duration(j.Minutes)).Unix()
	td.AccessUuid = uuid.New().String()

	td.RtExpires = time.Now().Add(time.Hour * 24 * 7).Unix()
	td.RefreshUuid = uuid.New().String()

	atClaims := gfjwt.NewClaim()
	atClaims.Set("authorized", j.Authorized)
	atClaims.Set("user_id", userID)
	atClaims.Set("permissions", permissions)
	atClaims.Set("access_uuid", td.AccessUuid)
	atClaims.Set("origin", origin)
	atClaims.Set("exp", time.Now().Add(time.Minute*time.Duration(j.Minutes)).Unix()) //JWT expires after 15mins
	td.AccessToken, err = j.algorithm.Encode(atClaims)
	if err != nil {
		return nil, err
	}

	//Creating Refresh Token
	rtClaims := gfjwt.NewClaim()
	rtClaims.Set("user_id", userID)
	rtClaims.Set("refresh_uuid", td.AccessUuid)
	rtClaims.Set("exp", time.Now().Add(time.Minute*time.Duration(j.Minutes)).Unix()) //JWT expires after 15mins
	td.RefreshToken, err = j.algorithm.Encode(rtClaims)
	if err != nil {
		return nil, err
	}

	return td, nil
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

	userID, _ := claims.Get("user_id")
	origin, _ := claims.Get("origin")
	permissions, _ := claims.Get("permissions")
	if r.Header.Get("Origin") != origin {
		return Token{}, errors.New("Unauthorized")
	}

	var token = Token{
		UserID:      userID.(int64),
		Permissions: permissions.([]string),
	}

	return token, nil
}
