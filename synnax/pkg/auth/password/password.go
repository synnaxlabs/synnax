package password

import (
	"github.com/cockroachdb/errors"
)

var (
	Invalid     = errors.New("[delta] - invalid credentials")
	InvalidHash = errors.New("[delta] - invalid hash")
)

// Raw represents a raw password. It is not safe to store the raw password on disk.
// The password should be hashed by calling Hash before saving it.
type Raw string

func (r Raw) Hash() (h Hashed, err error) {
	for _, hasher := range Hashers {
		h, err = hasher.Hash(r)
		if err == nil {
			return h, nil
		}
	}
	return h, errors.CombineErrors(InvalidHash, err)
}

// Hashed represents an encrypted hash of a password. It is safe to store the hash on disk.
// The hash can be compared against a raw password by calling Validate.
type Hashed []byte

func (h Hashed) Validate(r Raw) (err error) {
	for _, hasher := range Hashers {
		err = hasher.Compare(r, h)
		if err == nil {
			return nil
		}
	}
	return errors.CombineErrors(Invalid, err)
}
