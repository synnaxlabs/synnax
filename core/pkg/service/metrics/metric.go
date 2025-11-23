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
	// Create a cached disk size collector to avoid blocking metrics collection
	diskCollector := newCachedDiskCollector(dataPath)

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
			collect: diskCollector.collect,
		},
	}
}

// cachedDiskCollector calculates disk usage in the background to avoid blocking metrics collection
type cachedDiskCollector struct {
	dataPath string
	cached   float32
	done     chan struct{}
}

func newCachedDiskCollector(dataPath string) *cachedDiskCollector {
	c := &cachedDiskCollector{
		dataPath: dataPath,
		cached:   0,
		done:     make(chan struct{}),
	}

	// Start background updater
	go c.updateLoop()

	return c
}

func (c *cachedDiskCollector) collect() (float32, error) {
	// Return cached value immediately, never block
	return c.cached, nil
}

func (c *cachedDiskCollector) updateLoop() {
	// Update disk usage every 5 seconds
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Calculate initial value
	c.updateDiskSize()

	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			c.updateDiskSize()
		}
	}
}

func (c *cachedDiskCollector) updateDiskSize() {
	// In memory mode, dataPath is empty so keep 0
	if c.dataPath == "" {
		c.cached = 0
		return
	}

	size, err := calculateDirectorySize(c.dataPath)
	if err != nil {
		// On error, keep previous value
		return
	}

	// Convert bytes to megabytes and update cache
	c.cached = float32(size) / (1024 * 1024)
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
