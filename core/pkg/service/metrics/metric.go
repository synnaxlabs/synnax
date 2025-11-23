// Copyright 2025 Synnax Labs, Inc.
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
	"io/fs"
	"path/filepath"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

type metric struct {
	ch      channel.Channel
	collect func() (float32, error)
}

func allMetrics(dataPath string) []metric {
	return []metric{
		{
			ch: channel.Channel{
				Name:     "mem_percentage",
				DataType: telem.Float32T,
			},
			collect: func() (float32, error) {
				vm, err := mem.VirtualMemory()
				if err != nil {
					return 0, err
				}
				return float32(vm.UsedPercent), err
			},
		},
		{
			ch: channel.Channel{
				Name:     "cpu_percentage",
				DataType: telem.Float32T,
			},
			collect: func() (float32, error) {
				cpuUsage, err := cpu.Percent(0, false)
				if err != nil {
					return 0, err
				}
				if len(cpuUsage) < 1 {
					return 0, errors.New("no cpu usage metric found")
				}
				return float32(cpuUsage[0]), err
			},
		},
		{
			ch: channel.Channel{
				Name:     "disk_usage_mb",
				DataType: telem.Float32T,
			},
			collect: func() (float32, error) {
				// In memory mode, dataPath is empty so return 0
				if dataPath == "" {
					return 0, nil
				}
				size, err := calculateDirectorySize(dataPath)
				if err != nil {
					return 0, err
				}
				// Convert bytes to megabytes
				return float32(size) / (1024 * 1024), nil
			},
		},
	}
}

type sizeResult struct {
	size int64
	err  error
}

func calculateDirectorySize(path string) (int64, error) {
	// Create a context with timeout to prevent hanging on large directories
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	done := make(chan sizeResult, 1)

	go func() {
		defer func() {
			// Ensure we always send a result to prevent goroutine leaks
			select {
			case done <- sizeResult{}:
			default:
			}
		}()

		var size int64
		err := filepath.WalkDir(path, func(_ string, d fs.DirEntry, walkErr error) error {
			// Check if context is done to interrupt the walk
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if walkErr != nil {
				return walkErr
			}
			if !d.IsDir() {
				info, err := d.Info()
				if err != nil {
					return err
				}
				size += info.Size()
			}
			return nil
		})
		// Send result
		select {
		case done <- sizeResult{size: size, err: err}:
		case <-ctx.Done():
		}
	}()

	select {
	case <-ctx.Done():
		// Timeout - wait for goroutine
		<-done
		return 0, nil
	case result := <-done:
		if result.err == context.DeadlineExceeded {
			return 0, nil
		}
		return result.size, result.err
	}
}
