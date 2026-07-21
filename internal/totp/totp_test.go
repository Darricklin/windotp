package totp

import (
	"testing"
	"time"
)

func TestCodeRFC6238SHA1(t *testing.T) {
	// RFC 6238 uses 8 digits. The last six digits are the expected windotp code.
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	tests := []struct {
		unix int64
		want string
	}{
		{59, "287082"},
		{1111111109, "081804"},
		{1111111111, "050471"},
		{1234567890, "005924"},
		{2000000000, "279037"},
		{20000000000, "353130"},
	}
	for _, tt := range tests {
		got, err := Code(secret, time.Unix(tt.unix, 0))
		if err != nil {
			t.Fatal(err)
		}
		if got != tt.want {
			t.Errorf("Code(%d) = %s, want %s", tt.unix, got, tt.want)
		}
	}
}

func TestNormalizeSecret(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"jbsw y3dp ehpk 3pxp", "JBSWY3DPEHPK3PXP"},
		{"JBSWY3DPEHPK3PXP====", "JBSWY3DPEHPK3PXP"},
		{"otpauth://totp/JumpServer?secret=JBSWY3DPEHPK3PXP&issuer=JumpServer", "JBSWY3DPEHPK3PXP"},
	}
	for _, tt := range tests {
		got, err := NormalizeSecret(tt.input)
		if err != nil {
			t.Fatalf("NormalizeSecret(%q): %v", tt.input, err)
		}
		if got != tt.want {
			t.Errorf("NormalizeSecret(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNormalizeSecretRejectsInvalidInput(t *testing.T) {
	inputs := []string{
		"",
		"NOT-BASE32!",
		"otpauth://hotp/name?secret=JBSWY3DPEHPK3PXP",
		"otpauth://totp/name?secret=JBSWY3DPEHPK3PXP&algorithm=SHA256",
		"otpauth://totp/name?secret=JBSWY3DPEHPK3PXP&digits=8",
		"otpauth://totp/name?secret=JBSWY3DPEHPK3PXP&period=60",
	}
	for _, input := range inputs {
		if _, err := NormalizeSecret(input); err == nil {
			t.Errorf("NormalizeSecret(%q) unexpectedly succeeded", input)
		}
	}
}
