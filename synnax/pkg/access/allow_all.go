package access

type AllowAll struct{}

func (a AllowAll) Enforce(req Request) error { return nil }
