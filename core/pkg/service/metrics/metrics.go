// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package metrics

import (
	"context"
	"fmt"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

const (
	tsSizeMetricName = "ts_size_gb"
	kvSizeMetricName = "kv_size_gb"
)

type metric struct {
	collect func() (float32, error)
	ch      channel.Channel
}

func (s *Service) createMetrics(namePrefix string, idxKey channel.LocalKey) []metric {
	makeChannel := func(name string) channel.Channel {
		return channel.Channel{
			Name:       namePrefix + name,
			LocalIndex: idxKey,
			DataType:   telem.Float32T,
		}
	}
	metrics := []metric{
		{
			ch: makeChannel("mem_percentage"),
			collect: func() (float32, error) {
				vm, err := mem.VirtualMemory()
				if err != nil {
					return 0, err
				}
				return float32(vm.UsedPercent), nil
			},
		},
		{
			ch: makeChannel("cpu_percentage"),
			collect: func() (float32, error) {
				cpuUsage, err := cpu.Percent(0, false)
				if err != nil {
					return 0, err
				}
				if len(cpuUsage) < 1 {
					return 0, errors.New("no CPU usage metric found")
				}
				return float32(cpuUsage[0]), nil
			},
		},
		{
			ch: makeChannel(tsSizeMetricName),
			collect: func() (float32, error) {
				return float32(s.cfg.Storage.TSSize().Gigabytes()), nil
			},
		},
		{
			ch: makeChannel(kvSizeMetricName),
			collect: func() (float32, error) {
				return float32(s.cfg.Storage.KVSize().Gigabytes()), nil
			},
		},
	}
	return metrics
}

// createCalculatedMetrics creates, in the group with key g, each calculated metric
// whose output does not exist yet.
func createCalculatedMetrics(
	ctx context.Context,
	tx gorp.Tx,
	cfg ServiceConfig,
	namePrefix string,
	g *group.Key,
) error {
	name := namePrefix + "total_size_gb"
	exists, err := cfg.Channel.NewRetrieve().
		Where(channel.MatchNames(name)).
		Exists(ctx, tx)
	if err != nil || exists {
		return err
	}
	c := calculation.Calculation{Expression: fmt.Sprintf(
		"return %s%s + %s%s",
		namePrefix,
		tsSizeMetricName,
		namePrefix,
		kvSizeMetricName,
	)}
	_, err = cfg.Calculation.NewWriter(tx).CreateOutput(ctx, &c, name, g)
	return err
}
