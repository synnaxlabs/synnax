// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package backup

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/samber/lo"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
)

// File path conventions within the .sy archive. Used by both export and import
// to ensure consistent file layout.
const (
	ManifestPath  = "manifest.json"
	WorkspacePath = "workspaces/%s.json"
	UserPath      = "users/%s.json"
	DevicePath    = "devices/%s.json"
	TaskPath      = "tasks/%d.json"
	RangePath     = "ranges/%s.json"
	ChannelPath   = "channels/%d.json"
	LinePlotPath  = "workspaces/%s/lineplots/%s.json"
	SchematicPath = "workspaces/%s/schematics/%s.json"
	TablePath     = "workspaces/%s/tables/%s.json"
	LogPath       = "workspaces/%s/logs/%s.json"
	ArcPath       = "workspaces/%s/arcs/%s.json"
)

// ServiceConfig contains the dependencies needed by the backup service.
type ServiceConfig struct {
	Service      *service.Layer
	Distribution *distribution.Layer
}

// Service provides functionality for exporting and importing Synnax data
// as .sy archives.
type Service struct{ cfg ServiceConfig }

// NewService creates a new backup service with the provided configuration.
func NewService(cfg ServiceConfig) *Service { return &Service{cfg: cfg} }

// ExportRequest specifies which resources to include in an export.
type ExportRequest struct {
	WorkspaceKeys []uuid.UUID       `json:"workspace_keys"`
	UserKeys      []uuid.UUID       `json:"user_keys"`
	DeviceKeys    []string          `json:"device_keys"`
	TaskKeys      []task.Key        `json:"task_keys"`
	RangeKeys     []uuid.UUID       `json:"range_keys"`
	ChannelKeys   []distchannel.Key `json:"channel_keys"`
}

// OntologyIDs returns the ontology IDs for all resources referenced in the request.
func (r ExportRequest) OntologyIDs() []ontology.ID {
	ids := make([]ontology.ID, 0,
		len(r.WorkspaceKeys)+len(r.UserKeys)+len(r.DeviceKeys)+
			len(r.TaskKeys)+len(r.RangeKeys)+len(r.ChannelKeys))
	for _, k := range r.WorkspaceKeys {
		ids = append(ids, workspace.OntologyID(k))
	}
	for _, k := range r.UserKeys {
		ids = append(ids, user.OntologyID(k))
	}
	for _, k := range r.DeviceKeys {
		ids = append(ids, device.OntologyID(k))
	}
	for _, k := range r.TaskKeys {
		ids = append(ids, task.OntologyID(k))
	}
	for _, k := range r.RangeKeys {
		ids = append(ids, ranger.OntologyID(k))
	}
	for _, k := range r.ChannelKeys {
		ids = append(ids, distchannel.OntologyID(k))
	}
	return ids
}

// Export writes a .sy archive (ZIP format) to the provided writer containing all
// requested resources and their children.
func (s *Service) Export(ctx context.Context, req ExportRequest, w io.Writer) error {
	zw := zip.NewWriter(w)
	defer zw.Close()

	sections := make([]string, 0)

	if len(req.WorkspaceKeys) > 0 {
		if err := s.exportWorkspaces(ctx, req.WorkspaceKeys, zw); err != nil {
			return err
		}
		sections = append(sections, "workspaces")
	}
	if len(req.UserKeys) > 0 {
		if err := s.exportUsers(ctx, req.UserKeys, zw); err != nil {
			return err
		}
		sections = append(sections, "users")
	}
	if len(req.DeviceKeys) > 0 {
		if err := s.exportDevices(ctx, req.DeviceKeys, zw); err != nil {
			return err
		}
		sections = append(sections, "devices")
	}
	if len(req.TaskKeys) > 0 {
		if err := s.exportTasks(ctx, req.TaskKeys, zw); err != nil {
			return err
		}
		sections = append(sections, "tasks")
	}
	if len(req.RangeKeys) > 0 {
		if err := s.exportRanges(ctx, req.RangeKeys, zw); err != nil {
			return err
		}
		sections = append(sections, "ranges")
	}
	if len(req.ChannelKeys) > 0 {
		if err := s.exportChannels(ctx, req.ChannelKeys, zw); err != nil {
			return err
		}
		sections = append(sections, "channels")
	}

	return writeJSON(zw, ManifestPath, Manifest{
		Version:   Version,
		CreatedAt: time.Now().UTC(),
		Sections:  sections,
	})
}

func (s *Service) exportWorkspaces(ctx context.Context, keys []uuid.UUID, zw *zip.Writer) error {
	var workspaces []workspace.Workspace
	if err := s.cfg.Service.Workspace.NewRetrieve().
		WhereKeys(keys...).
		Entries(&workspaces).
		Exec(ctx, nil); err != nil {
		return errors.Wrap(err, "failed to retrieve workspaces")
	}
	for _, ws := range workspaces {
		if err := writeJSON(zw, fmt.Sprintf(WorkspacePath, ws.Key), newWorkspace(ws)); err != nil {
			return err
		}
		if err := s.exportWorkspaceChildren(ctx, ws, zw); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) exportWorkspaceChildren(ctx context.Context, ws workspace.Workspace, zw *zip.Writer) error {
	var children []ontology.Resource
	if err := s.cfg.Distribution.Ontology.NewRetrieve().
		WhereIDs(workspace.OntologyID(ws.Key)).
		TraverseTo(ontology.ChildrenTraverser).
		Entries(&children).
		Exec(ctx, nil); err != nil {
		return errors.Wrapf(err, "failed to retrieve children of workspace %s", ws.Key)
	}

	byType := lo.GroupBy(children, func(r ontology.Resource) ontology.Type { return r.ID.Type })
	wsKey := ws.Key.String()

	if ids, ok := byType[lineplot.OntologyType]; ok {
		keys, err := extractUUIDs(ids)
		if err != nil {
			return err
		}
		var entries []lineplot.LinePlot
		if err := s.cfg.Service.LinePlot.NewRetrieve().WhereKeys(keys...).Entries(&entries).Exec(ctx, nil); err != nil {
			return errors.Wrap(err, "failed to retrieve line plots")
		}
		for _, e := range entries {
			if err := writeJSON(zw, fmt.Sprintf(LinePlotPath, wsKey, e.Key), newDataVisualization(e.Name, e.Key, e.Data)); err != nil {
				return err
			}
		}
	}
	if ids, ok := byType[schematic.OntologyType]; ok {
		keys, err := extractUUIDs(ids)
		if err != nil {
			return err
		}
		var entries []schematic.Schematic
		if err := s.cfg.Service.Schematic.NewRetrieve().WhereKeys(keys...).Entries(&entries).Exec(ctx, nil); err != nil {
			return errors.Wrap(err, "failed to retrieve schematics")
		}
		for _, e := range entries {
			if err := writeJSON(zw, fmt.Sprintf(SchematicPath, wsKey, e.Key), newSchematic(e)); err != nil {
				return err
			}
		}
	}
	if ids, ok := byType[table.OntologyType]; ok {
		keys, err := extractUUIDs(ids)
		if err != nil {
			return err
		}
		var entries []table.Table
		if err := s.cfg.Service.Table.NewRetrieve().WhereKeys(keys...).Entries(&entries).Exec(ctx, nil); err != nil {
			return errors.Wrap(err, "failed to retrieve tables")
		}
		for _, e := range entries {
			if err := writeJSON(zw, fmt.Sprintf(TablePath, wsKey, e.Key), newDataVisualization(e.Name, e.Key, e.Data)); err != nil {
				return err
			}
		}
	}
	if ids, ok := byType[log.OntologyType]; ok {
		keys, err := extractUUIDs(ids)
		if err != nil {
			return err
		}
		var entries []log.Log
		if err := s.cfg.Service.Log.NewRetrieve().WhereKeys(keys...).Entries(&entries).Exec(ctx, nil); err != nil {
			return errors.Wrap(err, "failed to retrieve logs")
		}
		for _, e := range entries {
			if err := writeJSON(zw, fmt.Sprintf(LogPath, wsKey, e.Key), newDataVisualization(e.Name, e.Key, e.Data)); err != nil {
				return err
			}
		}
	}
	if ids, ok := byType[arc.OntologyType]; ok && s.cfg.Service.Arc != nil {
		keys, err := extractUUIDs(ids)
		if err != nil {
			return err
		}
		var entries []arc.Arc
		if err := s.cfg.Service.Arc.NewRetrieve().WhereKeys(keys...).Entries(&entries).Exec(ctx, nil); err != nil {
			return errors.Wrap(err, "failed to retrieve arcs")
		}
		for _, e := range entries {
			if err := writeJSON(zw, fmt.Sprintf(ArcPath, wsKey, e.Key), e); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) exportUsers(ctx context.Context, keys []uuid.UUID, zw *zip.Writer) error {
	var users []user.User
	if err := s.cfg.Service.User.NewRetrieve().WhereKeys(keys...).Entries(&users).Exec(ctx, nil); err != nil {
		return errors.Wrap(err, "failed to retrieve users")
	}
	for _, u := range users {
		if err := writeJSON(zw, fmt.Sprintf(UserPath, u.Key), u); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) exportDevices(ctx context.Context, keys []string, zw *zip.Writer) error {
	var devices []device.Device
	if err := s.cfg.Service.Device.NewRetrieve().WhereKeys(keys...).Entries(&devices).Exec(ctx, nil); err != nil {
		return errors.Wrap(err, "failed to retrieve devices")
	}
	for _, d := range devices {
		if err := writeJSON(zw, fmt.Sprintf(DevicePath, d.Key), d); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) exportTasks(ctx context.Context, keys []task.Key, zw *zip.Writer) error {
	var tasks []task.Task
	if err := s.cfg.Service.Task.NewRetrieve().WhereKeys(keys...).Entries(&tasks).Exec(ctx, nil); err != nil {
		return errors.Wrap(err, "failed to retrieve tasks")
	}
	for _, t := range tasks {
		if err := writeJSON(zw, fmt.Sprintf(TaskPath, t.Key), t); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) exportRanges(ctx context.Context, keys []uuid.UUID, zw *zip.Writer) error {
	var ranges []ranger.Range
	if err := s.cfg.Service.Ranger.NewRetrieve().WhereKeys(keys...).Entries(&ranges).Exec(ctx, nil); err != nil {
		return errors.Wrap(err, "failed to retrieve ranges")
	}
	for _, r := range ranges {
		if err := writeJSON(zw, fmt.Sprintf(RangePath, r.Key), r); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) exportChannels(ctx context.Context, keys []distchannel.Key, zw *zip.Writer) error {
	var channels []distchannel.Channel
	if err := s.cfg.Service.Channel.NewRetrieve().WhereKeys(keys...).Entries(&channels).Exec(ctx, nil); err != nil {
		return errors.Wrap(err, "failed to retrieve channels")
	}
	// Look up the parent group for each channel via the ontology.
	channelGroups := make(map[distchannel.Key]string, len(channels))
	for _, c := range channels {
		var parents []ontology.Resource
		if err := s.cfg.Distribution.Ontology.NewRetrieve().
			WhereIDs(distchannel.OntologyID(c.Key())).
			TraverseTo(ontology.ParentsTraverser).
			Entries(&parents).
			Exec(ctx, nil); err != nil {
			continue
		}
		for _, p := range parents {
			if p.ID.Type == group.OntologyType && p.Name != "Channels" {
				channelGroups[c.Key()] = p.Name
				break
			}
		}
	}
	for _, c := range channels {
		ch := newChannel(c)
		ch.Group = channelGroups[c.Key()]
		if err := writeJSON(zw, fmt.Sprintf(ChannelPath, c.Key()), ch); err != nil {
			return err
		}
	}
	return nil
}

func extractUUIDs(resources []ontology.Resource) ([]uuid.UUID, error) {
	keys := make([]uuid.UUID, 0, len(resources))
	for _, r := range resources {
		id, err := uuid.Parse(r.ID.Key)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse ontology key %q as UUID", r.ID.Key)
		}
		keys = append(keys, id)
	}
	return keys, nil
}

func writeJSON(zw *zip.Writer, path string, v any) error {
	w, err := zw.Create(path)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
