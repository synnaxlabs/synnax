// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package tree applies edits to the channel tree: the channel folders and the channels
// in them.
package tree

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening a tree service.
type ServiceConfig struct {
	// Group is the service of the channel folders.
	//
	// [REQUIRED]
	Group *group.Service
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Group = override.Nil(c.Group, other.Group)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("tree")
	v.NotNil("group", c.Group)
	return v.Error()
}

// Service applies edits to the channel tree.
type Service struct{ cfg ServiceConfig }

// OpenService opens a Service from cfgs.
func OpenService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{cfg: cfg}, nil
}

// DeleteFolders deletes the channel folders with keys in tx, as group.Writer.Delete
// deletes groups. A key that names no folder deletes nothing. A folder that does not
// hold channels fails the delete with a validation error, before any write.
func (s *Service) DeleteFolders(
	ctx context.Context,
	tx gorp.Tx,
	keys ...group.Key,
) error {
	var folders []group.Group
	if err := s.cfg.Group.NewRetrieve().
		Where(group.MatchKeys(keys...)).
		SkipMissingKeys().
		Entries(&folders).
		Exec(ctx, tx); err != nil {
		return err
	}
	for _, f := range folders {
		if f.Scope != resource.TypeChannel {
			return errors.Wrapf(
				validate.ErrValidation,
				"folder %s does not hold channels",
				f.Name,
			)
		}
	}
	return s.cfg.Group.NewWriter(tx).Delete(ctx, keys...)
}
