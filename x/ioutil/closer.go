package ioutil

type CloserFunc func() error

func (c CloserFunc) Close() error { return c() }
