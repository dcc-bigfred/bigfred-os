package auth

// Checker validates credentials and changes passwords.
type Checker interface {
	Authenticate(username, password string) error
	ChangePassword(username, current, newPassword string) error
}
