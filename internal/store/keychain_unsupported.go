//go:build !darwin

package store

import (
	"fmt"
	"runtime"
)

type unsupportedStore struct{}

func newPlatformStore() Store { return unsupportedStore{} }

func (unsupportedStore) Put(string, []byte) error   { return unsupportedError() }
func (unsupportedStore) Get(string) ([]byte, error) { return nil, unsupportedError() }
func (unsupportedStore) Delete(string) error        { return unsupportedError() }

func unsupportedError() error {
	return fmt.Errorf("macOS Keychain is unavailable on %s", runtime.GOOS)
}
