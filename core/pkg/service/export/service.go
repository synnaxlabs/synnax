// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package export

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
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

// ServiceConfig contains the dependencies needed by the export service.
type ServiceConfig struct {
	Ontology  *ontology.Ontology
	Workspace *workspace.Service
	LinePlot  *lineplot.Service
	Schematic *schematic.Service
	Table     *table.Service
	Arc       *arc.Service
	Log       *log.Service
	User      *user.Service
	Device    *device.Service
	Task      *task.Service
	Ranger    *ranger.Service
	Channel   *channel.Service
}

// Service provides functionality for exporting Synnax data into .syc archives.
type Service struct {
	cfg ServiceConfig
}

// NewService creates a new export service with the provided configuration.
func NewService(cfg ServiceConfig) *Service {
	return &Service{cfg: cfg}
}

// Request specifies which resources to include in the export.
type Request struct {
	WorkspaceKeys []uuid.UUID        `json:"workspace_keys"`
	UserKeys      []uuid.UUID        `json:"user_keys"`
	DeviceKeys    []string           `json:"device_keys"`
	TaskKeys      []task.Key         `json:"task_keys"`
	RangeKeys     []uuid.UUID        `json:"range_keys"`
	ChannelKeys   []distchannel.Key  `json:"channel_keys"`
	Path          string             `json:"path"`
}

// Export writes a .syc archive (ZIP format) to the provided writer containing all
// requested resources and their children.
func (s *Service) Export(ctx context.Context, req Request, w io.Writer) error {
	zw := zip.NewWriter(w)
	defer zw.Close()

	sections := make([]string, 0)

	if len(req.WorkspaceKeys) > 0 {
		if err := s.exportWorkspaces(ctx, req, zw); err != nil {
			return err
		}
		sections = append(sections, "workspaces")
	}

	if len(req.UserKeys) > 0 {
		if err := s.exportUsers(ctx, req, zw); err != nil {
			return err
		}
		sections = append(sections, "users")
	}

	if len(req.DeviceKeys) > 0 {
		if err := s.exportDevices(ctx, req, zw); err != nil {
			return err
		}
		sections = append(sections, "devices")
	}

	if len(req.TaskKeys) > 0 {
		if err := s.exportTasks(ctx, req, zw); err != nil {
			return err
		}
		sections = append(sections, "tasks")
	}

	if len(req.RangeKeys) > 0 {
		if err := s.exportRanges(ctx, req, zw); err != nil {
			return err
		}
		sections = append(sections, "ranges")
	}

	if len(req.ChannelKeys) > 0 {
		if err := s.exportChannels(ctx, req, zw); err != nil {
			return err
		}
		sections = append(sections, "channels")
	}

	return s.writeManifest(zw, sections)
}

func (s *Service) writeManifest(zw *zip.Writer, sections []string) error {
	manifest := Manifest{
		Version:   Version,
		CreatedAt: time.Now().UTC(),
		Sections:  sections,
	}
	return writeJSON(zw, "manifest.json", manifest)
}

func (s *Service) exportWorkspaces(
	ctx context.Context,
	req Request,
	zw *zip.Writer,
) error {
	var workspaces []workspace.Workspace
	if err := s.cfg.Workspace.NewRetrieve().
		WhereKeys(req.WorkspaceKeys...).
		Entries(&workspaces).
		Exec(ctx, nil); err != nil {
		return err
	}
	for _, ws := range workspaces {
		if err := writeJSON(
			zw,
			fmt.Sprintf("workspaces/%s.json", ws.Key),
			newWorkspace(ws),
		); err != nil {
			return err
		}
		if err := s.exportWorkspaceChildren(ctx, ws, zw); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) exportWorkspaceChildren(
	ctx context.Context,
	ws workspace.Workspace,
	zw *zip.Writer,
) error {
	var children []ontology.Resource
	if err := s.cfg.Ontology.NewRetrieve().
		WhereIDs(workspace.OntologyID(ws.Key)).
		TraverseTo(ontology.ChildrenTraverser).
		Entries(&children).
		Exec(ctx, nil); err != nil {
		return err
	}

	childrenByType := lo.GroupBy(children, func(r ontology.Resource) ontology.Type {
		return r.ID.Type
	})

	prefix := fmt.Sprintf("workspaces/%s", ws.Key)

	if ids, ok := childrenByType[lineplot.OntologyType]; ok {
		keys := extractUUIDs(ids)
		var entries []lineplot.LinePlot
		if err := s.cfg.LinePlot.NewRetrieve().
			WhereKeys(keys...).
			Entries(&entries).
			Exec(ctx, nil); err != nil {
			return err
		}
		for _, e := range entries {
			if err := writeJSON(
				zw,
				fmt.Sprintf("%s/lineplots/%s.json", prefix, e.Key),
				newLinePlot(e),
			); err != nil {
				return err
			}
		}
	}

	if ids, ok := childrenByType[schematic.OntologyType]; ok {
		keys := extractUUIDs(ids)
		var entries []schematic.Schematic
		if err := s.cfg.Schematic.NewRetrieve().
			WhereKeys(keys...).
			Entries(&entries).
			Exec(ctx, nil); err != nil {
			return err
		}
		for _, e := range entries {
			if err := writeJSON(
				zw,
				fmt.Sprintf("%s/schematics/%s.json", prefix, e.Key),
				newSchematic(e),
			); err != nil {
				return err
			}
		}
	}

	if ids, ok := childrenByType[table.OntologyType]; ok {
		keys := extractUUIDs(ids)
		var entries []table.Table
		if err := s.cfg.Table.NewRetrieve().
			WhereKeys(keys...).
			Entries(&entries).
			Exec(ctx, nil); err != nil {
			return err
		}
		for _, e := range entries {
			if err := writeJSON(
				zw,
				fmt.Sprintf("%s/tables/%s.json", prefix, e.Key),
				newTable(e),
			); err != nil {
				return err
			}
		}
	}

	if ids, ok := childrenByType[arc.OntologyType]; ok {
		keys := extractUUIDs(ids)
		var entries []arc.Arc
		if err := s.cfg.Arc.NewRetrieve().
			WhereKeys(keys...).
			Entries(&entries).
			Exec(ctx, nil); err != nil {
			return err
		}
		for _, e := range entries {
			if err := writeJSON(
				zw,
				fmt.Sprintf("%s/arcs/%s.json", prefix, e.Key),
				newArc(e),
			); err != nil {
				return err
			}
		}
	}

	if ids, ok := childrenByType[log.OntologyType]; ok {
		keys := extractUUIDs(ids)
		var entries []log.Log
		if err := s.cfg.Log.NewRetrieve().
			WhereKeys(keys...).
			Entries(&entries).
			Exec(ctx, nil); err != nil {
			return err
		}
		for _, e := range entries {
			if err := writeJSON(
				zw,
				fmt.Sprintf("%s/logs/%s.json", prefix, e.Key),
				newLog(e),
			); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Service) exportUsers(
	ctx context.Context,
	req Request,
	zw *zip.Writer,
) error {
	var users []user.User
	if err := s.cfg.User.NewRetrieve().
		WhereKeys(req.UserKeys...).
		Entries(&users).
		Exec(ctx, nil); err != nil {
		return err
	}
	for _, u := range users {
		if err := writeJSON(
			zw,
			fmt.Sprintf("users/%s.json", u.Key),
			newUser(u),
		); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) exportDevices(
	ctx context.Context,
	req Request,
	zw *zip.Writer,
) error {
	var devices []device.Device
	if err := s.cfg.Device.NewRetrieve().
		WhereKeys(req.DeviceKeys...).
		Entries(&devices).
		Exec(ctx, nil); err != nil {
		return err
	}
	for _, d := range devices {
		if err := writeJSON(
			zw,
			fmt.Sprintf("devices/%s.json", d.Key),
			newDevice(d),
		); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) exportTasks(
	ctx context.Context,
	req Request,
	zw *zip.Writer,
) error {
	var tasks []task.Task
	if err := s.cfg.Task.NewRetrieve().
		WhereKeys(req.TaskKeys...).
		Entries(&tasks).
		Exec(ctx, nil); err != nil {
		return err
	}
	for _, t := range tasks {
		if err := writeJSON(
			zw,
			fmt.Sprintf("tasks/%d.json", t.Key),
			newTask(t),
		); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) exportRanges(
	ctx context.Context,
	req Request,
	zw *zip.Writer,
) error {
	var ranges []ranger.Range
	if err := s.cfg.Ranger.NewRetrieve().
		WhereKeys(req.RangeKeys...).
		Entries(&ranges).
		Exec(ctx, nil); err != nil {
		return err
	}
	for _, r := range ranges {
		if err := writeJSON(
			zw,
			fmt.Sprintf("ranges/%s.json", r.Key),
			newRange(r),
		); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) exportChannels(
	ctx context.Context,
	req Request,
	zw *zip.Writer,
) error {
	var channels []distchannel.Channel
	if err := s.cfg.Channel.NewRetrieve().
		WhereKeys(req.ChannelKeys...).
		Entries(&channels).
		Exec(ctx, nil); err != nil {
		return err
	}
	for _, c := range channels {
		if err := writeJSON(
			zw,
			fmt.Sprintf("channels/%d.json", c.Key()),
			newChannel(c),
		); err != nil {
			return err
		}
	}
	return nil
}

func extractUUIDs(resources []ontology.Resource) []uuid.UUID {
	return lo.FilterMap(resources, func(r ontology.Resource, _ int) (uuid.UUID, bool) {
		id, err := uuid.Parse(r.ID.Key)
		return id, err == nil
	})
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
