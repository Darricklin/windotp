//go:build darwin

package store

import (
	"errors"
	"fmt"

	keychain "github.com/keybase/go-keychain"
)

type keychainStore struct{}

func newPlatformStore() Store { return keychainStore{} }

func (keychainStore) Put(profile string, secret []byte) error {
	item := keychain.NewGenericPassword(Service, profile, "WindOTP: "+profile, secret, "")
	item.SetSynchronizable(keychain.SynchronizableNo)
	item.SetAccessible(keychain.AccessibleWhenUnlocked)
	err := keychain.AddItem(item)
	if errors.Is(err, keychain.ErrorDuplicateItem) {
		query := keychain.NewItem()
		query.SetSecClass(keychain.SecClassGenericPassword)
		query.SetService(Service)
		query.SetAccount(profile)
		update := keychain.NewItem()
		update.SetData(secret)
		err = keychain.UpdateItem(query, update)
	}
	if err != nil {
		return fmt.Errorf("save secret to macOS Keychain: %w", err)
	}
	return nil
}

func (keychainStore) Get(profile string) ([]byte, error) {
	query := keychain.NewItem()
	query.SetSecClass(keychain.SecClassGenericPassword)
	query.SetService(Service)
	query.SetAccount(profile)
	query.SetMatchLimit(keychain.MatchLimitOne)
	query.SetReturnData(true)
	results, err := keychain.QueryItem(query)
	if errors.Is(err, keychain.ErrorItemNotFound) || (err == nil && len(results) == 0) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read secret from macOS Keychain: %w", err)
	}
	if len(results) != 1 || len(results[0].Data) == 0 {
		return nil, ErrNotFound
	}
	return results[0].Data, nil
}

func (keychainStore) Delete(profile string) error {
	err := keychain.DeleteGenericPasswordItem(Service, profile)
	if errors.Is(err, keychain.ErrorItemNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("delete secret from macOS Keychain: %w", err)
	}
	return nil
}
