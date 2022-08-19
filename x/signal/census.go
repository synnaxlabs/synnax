package signal

// Census allows the caller to examine the state of the signal context.
type Census interface {
	// Routines returns a slice of RoutineInfo for all routines forked by the
	// context.
	Routines() []RoutineInfo
}

// Routines implements the Census interface.
func (c *core) Routines() []RoutineInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.routines()
}

func (c *core) routines() []RoutineInfo {
	info := make([]RoutineInfo, len(c.mu.routines))
	for i, r := range c.mu.routines {
		info[i] = r.info()
	}
	return info
}
