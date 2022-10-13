

type Locator interface {
	Lookup(index channel.Key, position int) (rootposition int, err error)
	Close() error
}

type LocatorFactory interface {
	New(indexes ...channel.Key) (Locator, error)
}

