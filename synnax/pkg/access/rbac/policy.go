package rbac

import (
	"github.com/synnaxlabs/synnax/pkg/access"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/samber/lo"
)

type Policy struct {
	Subject ontology.ID
	Object  ontology.ID
	Actions []access.Action
	Effect  access.Effect
}

func NewPolicyKey(subject, object ontology.ID) string {
	return subject.String() + "-" + object.String()
}

// GorpKey implements the gorp.Entry interface.
func (p Policy) GorpKey() string { return NewPolicyKey(p.Subject, p.Object) }

// SetOptions implements the gorp.Entry interface.
func (p Policy) SetOptions() []interface{} { return nil }

// Matches returns true if the policy matches the given access.Request.
func (p Policy) Matches(req access.Request) bool {
	return p.Subject == req.Subject &&
		p.Object == req.Object &&
		lo.Contains(p.Actions, req.Action)
}
