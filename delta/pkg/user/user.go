package user

import (
	"github.com/google/uuid"
)

type User struct {
	Key      uuid.UUID `json:"key"`
	Username string    `json:"username"`
}

func (u User) GorpKey() uuid.UUID { return u.Key }

func (u User) SetOptions() []interface{} { return nil }
