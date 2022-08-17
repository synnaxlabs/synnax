package rbac

import (
	"github.com/arya-analytics/delta/pkg/access"
	"github.com/arya-analytics/x/query"
)

type Enforcer struct {
	DefaultEffect access.Effect
	Legislator    *Legislator
}

var _ access.Enforcer = (*Enforcer)(nil)

// Enforce implements the access.Enforcer interface.
func (e *Enforcer) Enforce(req access.Request) error {
	policies, err := e.Legislator.Retrieve(req.Subject, req.Object)
	if err != nil {
		if err == query.NotFound {
			return e.defaultErr()
		}
		return err
	}
	for _, p := range policies {
		if p.Matches(req) {
			if p.Effect == access.Deny {
				return access.Denied
			}
			return access.Granted
		}
	}
	return e.defaultErr()
}

func (e *Enforcer) defaultErr() error {
	if e.DefaultEffect == access.Deny {
		return access.Denied
	}
	return access.Granted
}
