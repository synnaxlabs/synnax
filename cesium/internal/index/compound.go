package index

import (
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/x/telem"
)

// CompoundSearcher is a collection of Seekers that are executed in order to implement
// the Searcher interface. The order of seekers is important:
//
//  1. The guess of the previous seeker is used as the guess of the next seeker.
//  2. The first seeker that returns a certain position is used as the result.
//  3. If no seeker returns a certain position, the seeker with the smallest uncertainty
//     is used as the result.
//  4. If any seeker returns an error, the entire operation is aborted.
//
// This generally means that lighter weight Seekers should be placed before heavier
// weight Seekers.
type CompoundSearcher []Searcher

var _ Searcher = CompoundSearcher{}

// SearchP implements Searcher.
func (c CompoundSearcher) SearchP(s telem.TimeStamp, guess position.Approximation) (position.Approximation, error) {
	for _, seeker := range c {
		approx, err := seeker.SearchP(s, guess)
		if err != nil {
			return position.Uncertain, err
		}
		if approx.Exact() {
			return approx, nil
		}
		if approx.BetterThan(guess) {
			guess = approx
		}
	}
	return guess, nil
}

// SearchTS implements Searcher.
func (c CompoundSearcher) SearchTS(s position.Position, guess telem.Approximation) (telem.Approximation, error) {
	for _, seeker := range c {
		approx, err := seeker.SearchTS(s, guess)
		if err != nil {
			return telem.Uncertain, err
		}
		if approx.Exact() {
			return approx, nil
		}
		if approx.Uncertainty() < guess.Uncertainty() {
			guess = approx
		}
	}
	return telem.Uncertain, nil
}

// Release implements Releaser.
func (c CompoundSearcher) Release() error {
	for _, seeker := range c {
		if err := seeker.Release(); err != nil {
			return err
		}
	}
	return nil
}

// CompoundWriter is a collection of Writers that are executed in order to implement
// the Writer interface.
type CompoundWriter []Writer

// Write implements Writer.
func (c CompoundWriter) Write(alignments []Alignment) (err error) {
	for _, w := range c {
		err = w.Write(alignments)
		if err != nil {
			return err
		}
	}
	return nil
}

// Release implements Releaser.
func (c CompoundWriter) Release() error {
	for _, w := range c {
		if err := w.Release(); err != nil {
			return err
		}
	}
	return nil
}
