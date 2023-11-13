package ranger

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/x/gorp"
)

type LabelAssignmentPolicy uint8

const (
	LabelAssignmentPolicyNone LabelAssignmentPolicy = iota
	LabelAssignmentPolicyUnique
)

type Label struct {
	Key              uuid.UUID             `json:"key" msgpack:"key"`
	Name             string                `json:"name" msgpack:"name"`
	Color            string                `json:"color" msgpack:"color"`
	AssignmentPolicy LabelAssignmentPolicy `json:"assignment_policy" msgpack:"assignment_policy"`
}

var _ gorp.Entry[uuid.UUID] = Label{}

func (l Label) GorpKey() uuid.UUID { return l.Key }

func (l Label) SetOptions() []interface{} { return nil }
