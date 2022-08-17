package version

// Counter is a simple monotonic counter that tracks a version.
type Counter int64

// Increment increments the Counter.
func (c Counter) Increment() Counter { return c + 1 }

// OlderThan returns true if the Counter is higher than other.
func (c Counter) OlderThan(other Counter) bool { return c > other }

// YoungerThan returns true if the counter is lower than other.
func (c Counter) YoungerThan(other Counter) bool { return c < other }

// EqualTo returns true if the counter is equal to another Counter.
func (c Counter) EqualTo(other Counter) bool { return c == other }
