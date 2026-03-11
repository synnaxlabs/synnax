package testutil

func MustSucceed[T any](value T, err error) T { return value }
