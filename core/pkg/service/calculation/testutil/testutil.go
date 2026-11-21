// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package testutil provides test helpers for working with calculations.
package testutil

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/telem"
)

// Calculation describes a calculation for a Creator to create.
type Calculation struct {
	// Name is the name of the output channel.
	Name string
	// Expression is the Arc expression that the calculation evaluates.
	Expression string
	// Operations are the operations of the calculation.
	Operations []channel.Operation
	// DataType is not read: the data type of Expression types the output.
	DataType telem.DataType
}

// Creator creates calculations for tests.
type Creator struct {
	// DB is the database that holds the channels.
	DB *gorp.DB
	// Channel is the channel service that stores the output channels.
	Channel *channel.Service
	// Calculation is the calculation service that stores the calculations.
	Calculation *calculation.Service
}

// Create creates calc and returns its output channel.
func (c Creator) Create(
	ctx context.Context,
	calc Calculation,
) (channel.Channel, error) {
	outputs, err := c.CreateMany(ctx, []Calculation{calc})
	if err != nil {
		return channel.Channel{}, err
	}
	return outputs[0], nil
}

// CreateMany creates calcs in one transaction and returns their output channels in the
// order of calcs.
func (c Creator) CreateMany(
	ctx context.Context,
	calcs []Calculation,
) ([]channel.Channel, error) {
	outputs := make([]channel.Channel, len(calcs))
	if err := c.DB.WithTx(ctx, func(tx gorp.Tx) error {
		w := c.Calculation.NewWriter(tx)
		for i, calc := range calcs {
			record := calculation.Calculation{
				Expression: calc.Expression,
				Operations: calculation.FromChannelOperations(calc.Operations),
			}
			created, err := w.CreateOutput(ctx, &record, calc.Name, nil)
			if err != nil {
				return err
			}
			outputs[i] = created[0]
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return outputs, nil
}

// CalculationOf returns the calculation that writes the channel with key output.
func (c Creator) CalculationOf(
	ctx context.Context,
	output channel.Key,
) (calculation.Calculation, error) {
	var calc calculation.Calculation
	err := c.Calculation.NewRetrieve().
		Where(calculation.MatchOutputs(output)).
		Entry(&calc).
		Exec(ctx, c.DB)
	return calc, err
}

// Apply applies acts to the output channel with key and writes its calculation, as the
// old channel API does.
func (c Creator) Apply(
	ctx context.Context,
	key channel.Key,
	acts ...channel.Action,
) error {
	return c.DB.WithTx(ctx, func(tx gorp.Tx) error {
		_, err := c.Calculation.NewWriter(tx).ApplyChannel(ctx, key, acts...)
		return err
	})
}

// Indexes returns the index channel of each output, in the order of outputs.
func (c Creator) Indexes(
	ctx context.Context,
	outputs []channel.Channel,
) ([]channel.Channel, error) {
	indexes := make([]channel.Channel, len(outputs))
	for i, output := range outputs {
		if err := c.Channel.NewRetrieve().
			Where(channel.MatchKeys(output.Index())).
			Entry(&indexes[i]).
			Exec(ctx, c.DB); err != nil {
			return nil, err
		}
	}
	return indexes, nil
}
