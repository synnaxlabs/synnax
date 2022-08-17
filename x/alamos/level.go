package alamos

import (
	"fmt"
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
	// Stringer returns a string representation of the filterTest.
	fmt.Stringer
}

// LevelFilterSet returns a LevelFilter than returns true if the Level is within the given set.
type LevelFilterSet []Level

func (e LevelFilterSet) Test(l2 Level) bool { return lo.Contains(e, l2) }

func (e LevelFilterSet) String() string {
	str := "LevelFilterSet{"
	for _, e := range e {
		str += e.String() + " "
	}
	return str + "}"
}

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

func (l LevelFilterThreshold) String() string {
	str := "LevelFilterThreshold{"
	if l.Below {
		str += "Below "
	} else {
		str += "Below "
	}
	return str + l.Level.String()
}

type LevelFilterAll struct{}

func (LevelFilterAll) Test(l Level) bool { return true }

func (LevelFilterAll) String() string { return "LevelFilterAll{}" }

type LevelFilterNone struct{}

func (LevelFilterNone) Test(l Level) bool { return false }

func (LevelFilterNone) String() string { return "LevelFilterNone{}" }
