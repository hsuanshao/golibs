package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/argon2"

	"github.com/hsuanshao/golibs/ctx"
)

type argon2Params struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	saltLength  uint32
	keyLength   uint32
}

// Use OWASP recommended parameter settings
// Memory: 64MB, Iterations: 1, Parallelism: 4
var defaultParams = &argon2Params{
	memory:      64 * 1024,
	iterations:  1,
	parallelism: 4,
	saltLength:  16,
	keyLength:   32,
}

// passwordUtility is the concrete implementation of PasswordUtility.
type passwordUtility struct {
	params *argon2Params
}

// New returns a PasswordUtility implementation with OWASP-recommended Argon2id parameters.
func New() PasswordUtility {
	return &passwordUtility{params: defaultParams}
}

var (
	ErrInvalidHash         = errors.New("the encoded hash is not in the correct format")
	ErrIncompatibleVersion = errors.New("incompatible version of argon2")
	ErrPasswordTooShort    = errors.New("password must be at least 8 characters long")
	ErrPasswordTooSimple   = errors.New("password must contain both letters and numbers")
	ErrInvalidSymbol       = errors.New("password contains invalid symbols")
	ErrSequenceFound       = errors.New("password cannot contain sequential characters")
	ErrAccountSimilar      = errors.New("password cannot contain part of your account name")
)

// CreateHash generates a password hash
// Returns format: $argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>
func (u *passwordUtility) CreateHash(ctx ctx.CTX, password string) (string, error) {
	p := u.params
	salt, err := generateRandomBytes(ctx, p.saltLength)
	if err != nil {
		ctx.WithField("err", err).Error("failed to generate random bytes")
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, p.iterations, p.memory, p.parallelism, p.keyLength)
	// Encode all parameters into standard PHC string format for future verification and upgrades
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)
	encodedHash := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.memory, p.iterations, p.parallelism, b64Salt, b64Hash)
	return encodedHash, nil
}

// ComparePasswordAndHash verifies the password
// Compares the input password with the previously stored hash string
func (u *passwordUtility) ComparePasswordAndHash(ctx ctx.CTX, password, encodedHash string) (bool, error) {
	// Parse parameters from the hash string (Salt, Memory, Iterations, etc.)
	p, salt, hash, err := decodeHash(ctx, encodedHash)
	if err != nil {
		ctx.WithField("err", err).Error("failed to decode hash")
		return false, err
	}
	// Hash the input password using the same parameters and Salt
	otherHash := argon2.IDKey([]byte(password), salt, p.iterations, p.memory, p.parallelism, p.keyLength)
	// Use ConstantTimeCompare to prevent timing attacks
	if subtle.ConstantTimeCompare(hash, otherHash) == 1 {
		ctx.Info("password matched")
		return true, nil
	}
	ctx.Info("password not matched")
	return false, nil
}

func generateRandomBytes(ctx ctx.CTX, n uint32) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		ctx.WithField("err", err).Error("failed to generate random bytes")
		return nil, err
	}
	return b, nil
}
func decodeHash(ctx ctx.CTX, encodedHash string) (params *argon2Params, salt, hash []byte, err error) {
	vals := strings.Split(encodedHash, "$")
	if len(vals) != 6 {
		ctx.WithField("encodedHash", encodedHash).Error("invalid hash format")
		return nil, nil, nil, ErrInvalidHash
	}
	var version int
	_, err = fmt.Sscanf(vals[2], "v=%d", &version)
	if err != nil {
		ctx.WithFields(logrus.Fields{"encodedHash": encodedHash, "err": err, "scan version": vals[2]}).Error("invalid hash format")
		return nil, nil, nil, err
	}
	if version != argon2.Version {
		ctx.WithFields(logrus.Fields{"encodedHash": encodedHash, "version": version, "argon2.Version": argon2.Version}).Error("incompatible version of argon2")
		return nil, nil, nil, ErrIncompatibleVersion
	}
	p := &argon2Params{}
	_, err = fmt.Sscanf(vals[3], "m=%d,t=%d,p=%d", &p.memory, &p.iterations, &p.parallelism)
	if err != nil {
		ctx.WithFields(logrus.Fields{"encodedHash": encodedHash, "err": err, "scan params": vals[3]}).Error("invalid hash format")
		return nil, nil, nil, err
	}
	salt, err = base64.RawStdEncoding.DecodeString(vals[4])
	if err != nil {
		ctx.WithFields(logrus.Fields{"encodedHash": encodedHash, "err": err, "scan salt": vals[4]}).Error("invalid hash format")
		return nil, nil, nil, err
	}
	p.saltLength = uint32(len(salt))
	hash, err = base64.RawStdEncoding.DecodeString(vals[5])
	if err != nil {
		ctx.WithFields(logrus.Fields{"encodedHash": encodedHash, "err": err, "scan hash": vals[5]}).Error("invalid hash format")
		return nil, nil, nil, err
	}
	p.keyLength = uint32(len(hash))
	return p, salt, hash, nil
}

// CheckPasswordComplexity verifies if the password meets security requirements
func (u *passwordUtility) CheckPasswordComplexity(ctx ctx.CTX, password, account string) error {
	if len(password) < 8 {
		ctx.Warn("input password length is less than 8")
		return ErrPasswordTooShort
	}

	hasLetter := false
	hasNumber := false
	allowedSymbols := "!_-@$#."

	for _, char := range password {
		switch {
		case char >= 'a' && char <= 'z':
			hasLetter = true
		case char >= 'A' && char <= 'Z':
			hasLetter = true
		case char >= '0' && char <= '9':
			hasNumber = true
		default:
			if !strings.ContainsRune(allowedSymbols, char) {
				ctx.Warn("input password contains invalid symbols")
				return ErrInvalidSymbol
			}
		}
	}

	if !hasLetter || !hasNumber {
		ctx.Warn("input password does not contain both letters and numbers")
		return ErrPasswordTooSimple
	}

	// Check for sequences (length 3 or more)
	// We check for sequential ASCII values.
	// Only checks letters and numbers sequences to avoid being too strict on symbols if desired,
	// but requirement says "No continuous letters and numbers".
	// Let's check for simple increasing/decreasing sequences of alphanumeric chars.

	lowerPwd := strings.ToLower(password)
	for i := 0; i < len(lowerPwd)-2; i++ {
		c1, c2, c3 := lowerPwd[i], lowerPwd[i+1], lowerPwd[i+2]
		if isAlphanumeric(c1) && isAlphanumeric(c2) && isAlphanumeric(c3) {
			if (c2 == c1+1 && c3 == c2+1) || (c2 == c1-1 && c3 == c2-1) {
				return ErrSequenceFound
			}
		}
	}

	// Password must not contain any 4-character substring of the login account name
	if len(account) >= 4 {
		lowerAccount := strings.ToLower(account)
		for i := 0; i <= len(lowerAccount)-4; i++ {
			sub := lowerAccount[i : i+4]
			if strings.Contains(strings.ToLower(password), sub) {
				ctx.Warn("input password contains any 4-character substring of the login account name")
				return ErrAccountSimilar
			}
		}
	}

	return nil
}

func isAlphanumeric(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z')
}
