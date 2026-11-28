// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package tree makes the edits of the channel tree: channels and the channel folders
// that hold them.
package tree

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening a Service.
type ServiceConfig struct {
	// Channel is the service of the channels in the tree.
	//
	// [REQUIRED]
	Channel *channel.Service
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Channel = override.Nil(c.Channel, other.Channel)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("channel.tree")
	v.NotNil("channel", c.Channel)
	return v.Error()
}

// Service makes the edits of the channel tree.
type Service struct{ cfg ServiceConfig }

// OpenService opens a Service with the provided configurations.
func OpenService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{cfg: cfg}, nil
}

// DeleteMembers deletes the tombstones of the channel folders groups in tx, so a delete
// of the folders sees only their live channels. It is the group.DeleteMembers hook of
// the channel scope.
func (s *Service) DeleteMembers(
	ctx context.Context,
	tx gorp.Tx,
	groups []group.Key,
) error {
	return s.cfg.Channel.NewWriter(tx).DeleteTombstones(ctx, groups...)
}
