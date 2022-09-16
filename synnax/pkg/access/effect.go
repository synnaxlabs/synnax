package access

type Effect uint8

const (
	Deny Effect = iota + 1
	Allow
)
