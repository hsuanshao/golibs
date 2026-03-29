package password

import "github.com/hsuanshao/golibs/ctx"

// PasswordUtility defines the interface for password hashing, verification, and complexity validation.
type PasswordUtility interface {
	// CreateHash generates an Argon2id password hash from the given plaintext password.
	// Returns the encoded hash string in PHC format: $argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>
	CreateHash(ctx ctx.CTX, password string) (string, error)

	// ComparePasswordAndHash verifies whether the given plaintext password matches the encoded hash.
	// Returns true if they match, false otherwise.
	ComparePasswordAndHash(ctx ctx.CTX, password, encodedHash string) (bool, error)

	// CheckPasswordComplexity validates the complexity requirements of the given password.
	// The account parameter is used to ensure the password does not contain account name substrings.
	// Returns a descriptive error if validation fails, or nil on success.
	CheckPasswordComplexity(ctx ctx.CTX, password, account string) error
}
