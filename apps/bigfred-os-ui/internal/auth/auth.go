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
	ErrInvalidCredentials   = errors.New("invalid_credentials")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrPasswordChangeFailed = errors.New("password_change_failed")
)

// Session is the JWT subject after login.
type Session struct {
	Username string `json:"username"`
}

// Config holds authentication settings.
type Config struct {
	PAMService string
	Username   string // dev / tests without PAM
	Password   string
	TTL        time.Duration
}

type claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// Service validates credentials and issues session JWTs.
type Service struct {
	checker   Checker
	jwtSecret []byte
	ttl       time.Duration
}

func newService(checker Checker, ttl time.Duration) (*Service, error) {
	if checker == nil {
		return nil, fmt.Errorf("credential checker is required")
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &Service{
		checker:   checker,
		jwtSecret: secret,
		ttl:       ttl,
	}, nil
}

// NewStatic creates a service with fixed credentials (unit tests).
func NewStatic(username, password string, ttl time.Duration) (*Service, error) {
	return newService(NewStaticChecker(username, password), ttl)
}

func (s *Service) Login(username, password string) (Session, error) {
	if err := s.checker.Authenticate(username, password); err != nil {
		return Session{}, err
	}
	return Session{Username: username}, nil
}

func (s *Service) ChangePassword(username, current, newPassword string) error {
	return s.checker.ChangePassword(username, current, newPassword)
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
