+++
description = "Go functions return zero values with an error"
paths = ["**/*.go"]
+++

When a Go function returns an error, its other return values are zero values: `0`,
`nil`, `""`, or an empty struct. Report a partial or stale value returned next to a
non-nil error. The exception is a contract that defines the partial value, such as the
byte count of `io.Reader` and `io.Writer`: do not report it.
