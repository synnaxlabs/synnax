// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mqtt

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"sync"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/sparkplug"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

const (
	// browseSparkplugCommand lists the edge nodes of a broker device, or the tags of
	// one edge node.
	browseSparkplugCommand = "browse_sparkplug"
	// birthQuiet is the silence after a node birth that ends a browse of its tags. The
	// births of the devices follow the birth of their edge node at once.
	birthQuiet = 500 * time.Millisecond
)

type browseSparkplugArgs struct {
	// Device is the key of the broker device to browse.
	Device device.Key `json:"device"`
	// Group limits a browse for edge nodes to one group ID.
	Group string `json:"group"`
	// EdgeNode selects a browse for the tags of this edge node of Group. Empty selects
	// a browse for edge nodes.
	EdgeNode string `json:"edge_node"`
	// Duration is the longest time to listen for, in milliseconds. Zero selects 3 s.
	Duration int `json:"duration"`
}

// browsedNode is one edge node that a browse heard.
type browsedNode struct {
	Group    string   `json:"group"`
	EdgeNode string   `json:"edge_node"`
	Devices  []string `json:"devices"`
}

// browsedTag is one tag of a birth message.
type browsedTag struct {
	// Device is the device ID. Empty for a tag of the edge node.
	Device string `json:"device"`
	Name   string `json:"name"`
	// DataType is the config form of the Sparkplug B data type, or its name in the
	// specification for a type with no config form.
	DataType string `json:"data_type"`
	// Value is the text of the value in the birth message.
	Value string `json:"value"`
	// Supported is true when a read task can take the values of the tag.
	Supported bool `json:"supported"`
}

type browseSparkplugResult struct {
	// Nodes holds the edge nodes of a browse for edge nodes.
	Nodes []browsedNode `json:"nodes"`
	// Tags holds the tags of a browse of one edge node.
	Tags []browsedTag `json:"tags"`
}

func (s *scanner) browseSparkplug(
	ctx context.Context,
	cmd task.Command,
) (browseSparkplugResult, error) {
	var (
		args browseSparkplugArgs
		res  = browseSparkplugResult{Nodes: []browsedNode{}, Tags: []browsedTag{}}
	)
	if err := cmd.Args.Unmarshal(&args); err != nil {
		return res, errors.Wrapf(
			validate.ErrValidation, "invalid arguments: %s", err.Error(),
		)
	}
	dev, cfg, err := s.browseConfig(ctx, args.Device)
	if err != nil {
		return res, err
	}
	duration := browseDuration(args.Duration)
	if args.EdgeNode == "" {
		res.Nodes, err = browseNodes(ctx, cfg, nodeFilters(dev, args.Group), duration)
		return res, err
	}
	node := sparkplug.NodeID{Group: args.Group, EdgeNode: args.EdgeNode}
	if err = node.Validate(); err != nil {
		return res, err
	}
	res.Tags, err = browseTags(ctx, cfg, node, duration)
	return res, err
}

// nodeFilters returns the topic filters of a browse for edge nodes: the group that
// the user gave, else the groups in the properties of dev, else every group.
func nodeFilters(dev device.Device, group string) []string {
	groups := []string{group}
	if group == "" {
		var props Properties
		// newClientConfig decoded the properties before, so they decode now.
		_ = dev.Properties.Unmarshal(&props)
		groups = props.Sparkplug.Groups
	}
	if len(groups) == 0 {
		return []string{sparkplug.Namespace + "/#"}
	}
	filters := make([]string, 0, len(groups))
	for _, g := range groups {
		filters = append(filters, sparkplug.Namespace+"/"+g+"/#")
	}
	return filters
}

// browseNodes lists the edge nodes and devices that publish during duration. An edge
// node reports by exception, so a quiet one does not appear.
func browseNodes(
	ctx context.Context,
	cfg clientConfig,
	filters []string,
	duration time.Duration,
) ([]browsedNode, error) {
	var (
		mu    sync.Mutex
		nodes = make(map[sparkplug.NodeID]set.Set[string])
	)
	opts := newClientOptions(cfg)
	opts.SetDefaultPublishHandler(func(_ paho.Client, m paho.Message) {
		topic, err := sparkplug.ParseTopic(m.Topic())
		if err != nil || topic.Type == sparkplug.State {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if nodes[topic.Node] == nil {
			nodes[topic.Node] = make(set.Set[string])
		}
		if topic.Device != "" {
			nodes[topic.Node].Add(topic.Device)
		}
	})
	client, disconnect, err := open(ctx, opts)
	if err != nil {
		return nil, err
	}
	defer disconnect()
	subs := make(map[string]byte, len(filters))
	for _, f := range filters {
		subs[f] = 0
	}
	if err = wait(ctx, client.SubscribeMultiple(subs, nil)); err != nil {
		return nil, err
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
	}
	mu.Lock()
	defer mu.Unlock()
	res := make([]browsedNode, 0, len(nodes))
	for id, devices := range nodes {
		n := browsedNode{
			Group: id.Group, EdgeNode: id.EdgeNode, Devices: make([]string, 0),
		}
		for d := range devices {
			n.Devices = append(n.Devices, d)
		}
		slices.Sort(n.Devices)
		res = append(res, n)
	}
	sort.Slice(res, func(i, j int) bool {
		if res[i].Group != res[j].Group {
			return res[i].Group < res[j].Group
		}
		return res[i].EdgeNode < res[j].EdgeNode
	})
	return res, nil
}

// browseTags asks node for a rebirth and lists the tags of the births that follow.
func browseTags(
	ctx context.Context,
	cfg clientConfig,
	node sparkplug.NodeID,
	duration time.Duration,
) ([]browsedTag, error) {
	var (
		host   = sparkplug.NewHost()
		births = make(chan sparkplug.Event, 16)
	)
	opts := newClientOptions(cfg).SetOrderMatters(true)
	opts.SetDefaultPublishHandler(func(_ paho.Client, m paho.Message) {
		topic, err := sparkplug.ParseTopic(m.Topic())
		if err != nil {
			return
		}
		// Only the delivery goroutine of the client uses host.
		ev, err := host.Handle(topic, m.Payload())
		if err != nil || (ev.Type != sparkplug.NBirth && ev.Type != sparkplug.DBirth) {
			return
		}
		select {
		case births <- ev:
		default:
		}
	})
	client, disconnect, err := open(ctx, opts)
	if err != nil {
		return nil, err
	}
	defer disconnect()
	filters := make(map[string]byte)
	for _, filter := range node.Filters() {
		filters[filter] = 0
	}
	if err = wait(ctx, client.SubscribeMultiple(filters, nil)); err != nil {
		return nil, err
	}
	payload, err := sparkplug.EncodeRebirth(telem.Now())
	if err != nil {
		return nil, err
	}
	topic := sparkplug.CommandTopic(node, "").String()
	if err = wait(ctx, client.Publish(topic, 0, false, payload)); err != nil {
		return nil, err
	}
	var (
		tags  = make(map[tagID]browsedTag)
		born  = false
		timer = time.NewTimer(duration)
	)
	defer timer.Stop()
listen:
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
			break listen
		case ev := <-births:
			if ev.Type == sparkplug.NBirth {
				born = true
				clear(tags)
			}
			for _, m := range ev.Metrics {
				if m.Name == sparkplug.BdSeqMetric {
					continue
				}
				id := tagID{node: node, device: ev.Device, name: m.Name}
				tags[id] = newBrowsedTag(ev.Device, m)
			}
			if born {
				timer.Reset(birthQuiet)
			}
		}
	}
	if !born {
		return nil, errors.Newf(
			"edge node %s did not answer the rebirth request", node,
		)
	}
	res := make([]browsedTag, 0, len(tags))
	for _, t := range tags {
		res = append(res, t)
	}
	sort.Slice(res, func(i, j int) bool {
		if res[i].Device != res[j].Device {
			return res[i].Device < res[j].Device
		}
		return res[i].Name < res[j].Name
	})
	return res, nil
}

func newBrowsedTag(device string, m sparkplug.Metric) browsedTag {
	t := browsedTag{
		Device:    device,
		Name:      m.Name,
		DataType:  m.DataType.String(),
		Supported: m.DataType.Supported(),
	}
	for name, dt := range sparkplugTypes {
		if dt == m.DataType {
			t.DataType = string(name)
		}
	}
	if m.Value != nil {
		t.Value = fmt.Sprint(m.Value)
	}
	return t
}
