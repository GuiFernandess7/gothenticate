package pkg

import "golang.org/x/crypto/bcrypt"

// DummyHash is a valid bcrypt hash with no real password behind it.
// Compare against it when a lookup finds no user, so CheckPassword
// always pays the same bcrypt cost whether or not the account exists
// -- otherwise "user not found" responds measurably faster than
// "wrong password" and leaks which emails are registered.
var DummyHash string

func init() {
	hash, err := bcrypt.GenerateFromPassword(
		[]byte("dummy-password-for-timing-safety"),
		bcrypt.DefaultCost,
	)
	if err != nil {
		panic(err)
	}
	DummyHash = string(hash)
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	return string(hash), err
}

func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword(
		[]byte(hash),
		[]byte(password),
	)
	return err == nil
}
