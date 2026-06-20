package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const SessionCookieName = "bigfred_os_session"

var (
	ErrInvalidCredentials = errors.New("invalid_credentials")
	ErrUnauthorized       = errors.New("unauthorized")
)

// Session is the JWT subject after login.
type Session struct {
	Username string `json:"username"`
}

type claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// Service validates CLI-configured credentials and issues session JWTs.
type Service struct {
	username  string
	password  string
	jwtSecret []byte
	ttl       time.Duration
}

func New(username, password string, ttl time.Duration) (*Service, error) {
	if username == "" || password == "" {
		return nil, fmt.Errorf("username and password are required")
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &Service{
		username:  username,
		password:  password,
		jwtSecret: secret,
		ttl:       ttl,
	}, nil
}

func (s *Service) Login(username, password string) (Session, error) {
	if username != s.username || password != s.password {
		return Session{}, ErrInvalidCredentials
	}
	return Session{Username: s.username}, nil
}

func (s *Service) IssueToken(sess Session) (token string, expires time.Time, err error) {
	expires = time.Now().Add(s.ttl)
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		Username: sess.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expires),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        randomID(),
		},
	})
	token, err = t.SignedString(s.jwtSecret)
	return token, expires, err
}

func (s *Service) VerifyToken(token string) (Session, error) {
	parsed, err := jwt.ParseWithClaims(token, &claims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return Session{}, ErrUnauthorized
	}
	c, ok := parsed.Claims.(*claims)
	if !ok || !parsed.Valid || c.Username == "" {
		return Session{}, ErrUnauthorized
	}
	return Session{Username: c.Username}, nil
}

func (s *Service) SessionTTL() time.Duration {
	return s.ttl
}

func randomID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
