package auth

// StaticChecker compares credentials in memory (tests and local dev without PAM).
type StaticChecker struct {
	username string
	password string
}

func NewStaticChecker(username, password string) *StaticChecker {
	return &StaticChecker{username: username, password: password}
}

func (c *StaticChecker) Authenticate(username, password string) error {
	if username != c.username || password != c.password {
		return ErrInvalidCredentials
	}
	return nil
}

func (c *StaticChecker) ChangePassword(username, current, newPassword string) error {
	if err := c.Authenticate(username, current); err != nil {
		return err
	}
	if newPassword == "" {
		return ErrPasswordChangeFailed
	}
	c.password = newPassword
	return nil
}
