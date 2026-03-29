package password

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/hsuanshao/golibs/ctx"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func newTestCTX() ctx.CTX {
	logger := logrus.New()
	logger.Out = io.Discard
	return ctx.CTX{
		Context:     context.Background(),
		FieldLogger: logger,
	}
}

func TestCreateHash(t *testing.T) {
	c := newTestCTX()
	u := New()
	password := "secret123"
	hash, err := u.CreateHash(c, password)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.True(t, strings.HasPrefix(hash, "$argon2id$"))
}

func TestComparePasswordAndHash_Success(t *testing.T) {
	c := newTestCTX()
	u := New()
	password := "correct_password"
	hash, err := u.CreateHash(c, password)
	assert.NoError(t, err)

	match, err := u.ComparePasswordAndHash(c, password, hash)
	assert.NoError(t, err)
	assert.True(t, match)
}

func TestComparePasswordAndHash_WrongPassword(t *testing.T) {
	c := newTestCTX()
	u := New()
	password := "my_password"
	hash, err := u.CreateHash(c, password)
	assert.NoError(t, err)

	match, err := u.ComparePasswordAndHash(c, "wrong_password", hash)
	assert.NoError(t, err)
	assert.False(t, match)
}

func TestComparePasswordAndHash_InvalidHashFormat(t *testing.T) {
	c := newTestCTX()
	u := New()
	// Not enough parts
	match, err := u.ComparePasswordAndHash(c, "password", "invalid$format")
	assert.ErrorIs(t, err, ErrInvalidHash)
	assert.False(t, match)

	// Wrong version format part (unable to parse integer)
	match, err = u.ComparePasswordAndHash(c, "password", "$argon2id$v=invalid$m=65536,t=1,p=4$salt$hash")
	assert.Error(t, err)
	assert.False(t, match)
}

func TestComparePasswordAndHash_IncompatibleVersion(t *testing.T) {
	c := newTestCTX()
	u := New()
	// Construct a hash with wrong version
	// Current version is 19 (0x13)
	// Format: $argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s
	hash := "$argon2id$v=99$m=65536,t=1,p=4$c2FsdA$aGFzaA"

	match, err := u.ComparePasswordAndHash(c, "password", hash)
	assert.ErrorIs(t, err, ErrIncompatibleVersion)
	assert.False(t, match)
}

func TestComparePasswordAndHash_InvalidParams(t *testing.T) {
	c := newTestCTX()
	u := New()
	// corrupted numeric params
	hash := "$argon2id$v=19$m=abc,t=1,p=4$c2FsdA$aGFzaA"
	match, err := u.ComparePasswordAndHash(c, "password", hash)
	assert.Error(t, err)
	assert.False(t, match)
}

func TestComparePasswordAndHash_InvalidBase64(t *testing.T) {
	c := newTestCTX()
	u := New()
	// Invalid salt base64
	hash := "$argon2id$v=19$m=65536,t=1,p=4$!@#$aGFzaA"
	match, err := u.ComparePasswordAndHash(c, "password", hash)
	assert.Error(t, err)
	assert.False(t, match)

	// Invalid hash base64
	hash2 := "$argon2id$v=19$m=65536,t=1,p=4$c2FsdA$!@#"
	match, err2 := u.ComparePasswordAndHash(c, "password", hash2)
	assert.Error(t, err2)
	assert.False(t, match)
}

func TestGenerateRandomBytes(t *testing.T) {
	c := newTestCTX()
	// This is an internal function but useful to test indirectly or directly if exported.
	// Since it is unexported, we can test it because we are in the same package (whitebox testing).
	b, err := generateRandomBytes(c, 16)
	assert.NoError(t, err)
	assert.Len(t, b, 16)

	b2, err := generateRandomBytes(c, 16)
	assert.NoError(t, err)
	assert.NotEqual(t, b, b2, "Random bytes should practically never be equal")
}

func TestCheckPasswordComplexity(t *testing.T) {
	c := newTestCTX()
	u := New()
	account := "userone"

	tests := []struct {
		name     string
		password string
		account  string
		wantErr  error
	}{
		{
			name:     "Valid Password",
			password: "Valid1@Password",
			account:  account,
			wantErr:  nil,
		},
		{
			name:     "Too Short",
			password: "Short1!",
			account:  account,
			wantErr:  ErrPasswordTooShort,
		},
		{
			name:     "No Number",
			password: "NoNumberPassword!",
			account:  account,
			wantErr:  ErrPasswordTooSimple,
		},
		{
			name:     "No Letter",
			password: "12345678!",
			account:  account,
			wantErr:  ErrPasswordTooSimple,
		},
		{
			name:     "Invalid Symbol",
			password: "Valid1%Password",
			account:  account,
			wantErr:  ErrInvalidSymbol,
		},
		{
			name:     "Sequence ABC",
			password: "abc1@Password",
			account:  account,
			wantErr:  ErrSequenceFound,
		},
		{
			name:     "Sequence 123",
			password: "123Valid@Password",
			account:  account,
			wantErr:  ErrSequenceFound,
		},
		{
			name:     "Reverse Sequence cba",
			password: "cba1@Password",
			account:  account,
			wantErr:  ErrSequenceFound,
		},
		{
			name:     "Account Similarity",
			password: "Valid1@userone",
			account:  account,
			wantErr:  ErrAccountSimilar,
		},
		{
			name:     "Account Similarity Substring",
			password: "Valid1@seroPassword", // "sero" is part of "userone"
			account:  account,
			wantErr:  ErrAccountSimilar,
		},
		{
			name:     "Safe Substring Account",
			password: "Valid1@serPassword", // "ser" is "userone"[1:4], length 3, should be safe
			account:  account,
			wantErr:  nil,
		},
		{
			name:     "Allowed Symbols",
			password: "Valid!_@$#.Password1",
			account:  account,
			wantErr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := u.CheckPasswordComplexity(c, tt.password, tt.account)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr, "Password: %s", tt.password)
			} else {
				assert.NoError(t, err, "Password: %s", tt.password)
			}
		})
	}
}
