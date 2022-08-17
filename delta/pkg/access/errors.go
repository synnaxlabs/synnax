package access

import (
	"github.com/cockroachdb/errors"
)

var (
	Denied        = errors.New("[access] - forbidden")
	Granted error = nil
)
