package medication

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/x/gorp"
)

//go:generate jerky
type Medication struct {
	Key      uuid.UUID `json:"key"`
	Name     string    `json:"name"`
	Duration int       `json:"duration"`
}

var _ gorp.Entry[uuid.UUID] = Medication{}

func (m Medication) GorpKey() uuid.UUID { return m.Key }

func (m Medication) SetOptions() []any { return nil }
