package alamos

import (
	"github.com/samber/lo"
)

//go:generate stringer -type=Level
type Level uint8

const (
	Debug Level = iota
	Perf
	Production
)

// LevelFilter is used to filterTest out certain metrics, reports, or experiments based on a configured Level.
type LevelFilter interface {
	Test(Level) bool
}

// LevelFilterSet returns a LevelFilter than returns true if the Level is within the given set.
type LevelFilterSet []Level

func (e LevelFilterSet) Test(l2 Level) bool { return lo.Contains(e, l2) }

type LevelFilterThreshold struct {
	Below bool
	Level Level
}

func (l LevelFilterThreshold) Test(l2 Level) bool {
	if l.Below {
		return l2 <= l.Level
	}
	return l2 >= l.Level
}

type LevelFilterAll struct{}

func (LevelFilterAll) Test(l Level) bool { return true }
