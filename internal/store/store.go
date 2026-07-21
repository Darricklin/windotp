package store

import "errors"

const Service = "dev.windotp.totp"

var ErrNotFound = errors.New("TOTP secret not found in Keychain")

type Store interface {
	Put(profile string, secret []byte) error
	Get(profile string) ([]byte, error)
	Delete(profile string) error
}

func New() Store {
	return newPlatformStore()
}
