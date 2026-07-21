package totp

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const Period = 30 * time.Second

func NormalizeSecret(input string) (string, error) {
	input = strings.TrimSpace(input)
	if strings.HasPrefix(strings.ToLower(input), "otpauth://") {
		u, err := url.Parse(input)
		if err != nil {
			return "", fmt.Errorf("parse otpauth URL: %w", err)
		}
		if !strings.EqualFold(u.Host, "totp") {
			return "", fmt.Errorf("only otpauth TOTP URLs are supported")
		}
		query := u.Query()
		if algorithm := query.Get("algorithm"); algorithm != "" && !strings.EqualFold(algorithm, "SHA1") {
			return "", fmt.Errorf("only SHA1 otpauth URLs are supported")
		}
		if digits := query.Get("digits"); digits != "" && digits != "6" {
			return "", fmt.Errorf("only 6-digit otpauth URLs are supported")
		}
		if period := query.Get("period"); period != "" && period != "30" {
			return "", fmt.Errorf("only 30-second otpauth URLs are supported")
		}
		input = query.Get("secret")
		if input == "" {
			return "", fmt.Errorf("otpauth URL has no secret")
		}
	}

	input = strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '\r', '\n':
			return -1
		default:
			return r
		}
	}, input)
	secret := strings.ToUpper(strings.TrimRight(input, "="))
	if secret == "" {
		return "", fmt.Errorf("secret is empty")
	}
	if _, err := decode(secret); err != nil {
		return "", fmt.Errorf("invalid Base32 secret: %w", err)
	}
	return secret, nil
}

func Code(secret string, at time.Time) (string, error) {
	key, err := decode(secret)
	if err != nil {
		return "", fmt.Errorf("decode Base32 secret: %w", err)
	}

	var counter [8]byte
	binary.BigEndian.PutUint64(counter[:], uint64(at.Unix()/int64(Period/time.Second)))
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(counter[:])
	digest := mac.Sum(nil)
	offset := digest[len(digest)-1] & 0x0f
	value := binary.BigEndian.Uint32(digest[offset:offset+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", value%1_000_000), nil
}

func Remaining(at time.Time) time.Duration {
	period := int64(Period)
	elapsed := at.UnixNano() % period
	return time.Duration(period - elapsed)
}

func decode(secret string) ([]byte, error) {
	clean := strings.ToUpper(strings.TrimRight(strings.TrimSpace(secret), "="))
	return base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(clean)
}
