package ferrors

type Type string

const (
	Empty   = Type("")
	Nil     = Type("nil")
	Unknown = Type("unknown")
	Roach   = Type("roach")
)

type Error interface {
	error
	// FreighterType returns the type of the error. Freighter uses this to determine the
	// correct decode to use on the other end of the transport.
	FreighterType() Type
}

func WithType(err error, t Type) error {
	if err == nil {
		return nil
	}
	return &typed{error: err, t: t}
}

type typed struct {
	error
	t Type
}

var _ Error = (*typed)(nil)

func (t typed) FreighterType() Type { return t.t }
