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
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/samber/lo"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
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

// Analyze reads a .sy archive and compares each entity against existing data,
// returning an analysis of what would happen if the archive were imported.
func (s *Service) Analyze(ctx context.Context, r io.ReaderAt, size int64) (AnalyzeResponse, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return AnalyzeResponse{}, errors.Wrap(err, "failed to open archive")
	}
	var items []AnalysisItem
	manifest, err := readArchiveJSON[Manifest](zr, ManifestPath)
	if err != nil {
		return AnalyzeResponse{}, err
	}
	for _, section := range manifest.Sections {
		var sectionItems []AnalysisItem
		switch section {
		case "channels":
			sectionItems, err = s.analyzeChannels(ctx, zr)
		case "workspaces":
			sectionItems, err = s.analyzeWorkspaces(ctx, zr)
		case "users":
			sectionItems, err = s.analyzeUsers(ctx, zr)
		case "devices":
			sectionItems, err = s.analyzeDevices(ctx, zr)
		case "tasks":
			sectionItems, err = s.analyzeTasks(ctx, zr)
		case "ranges":
			sectionItems, err = s.analyzeRanges(ctx, zr)
		}
		if err != nil {
			return AnalyzeResponse{}, err
		}
		items = append(items, sectionItems...)
	}
	return AnalyzeResponse{Items: items}, nil
}

// Import reads a .sy archive and creates or updates entities according to the
// conflict resolution policies in the request.
func (s *Service) Import(ctx context.Context, r io.ReaderAt, size int64, req ImportRequest) (ImportResponse, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return ImportResponse{}, errors.Wrap(err, "failed to open archive")
	}
	manifest, err := readArchiveJSON[Manifest](zr, ManifestPath)
	if err != nil {
		return ImportResponse{}, err
	}
	var resp ImportResponse
	channelRemap := make(map[distchannel.Key]distchannel.Key)

	if lo.Contains(manifest.Sections, "channels") {
		if err := s.importChannels(ctx, zr, req, channelRemap, &resp); err != nil {
			return resp, err
		}
	}
	if lo.Contains(manifest.Sections, "ranges") {
		if err := s.importRanges(ctx, zr, req, &resp); err != nil {
			return resp, err
		}
	}
	if lo.Contains(manifest.Sections, "users") {
		if err := s.importUsers(ctx, zr, req, &resp); err != nil {
			return resp, err
		}
	}
	if lo.Contains(manifest.Sections, "devices") {
		if err := s.importDevices(ctx, zr, req, &resp); err != nil {
			return resp, err
		}
	}
	if lo.Contains(manifest.Sections, "tasks") {
		if err := s.importTasks(ctx, zr, req, &resp); err != nil {
			return resp, err
		}
	}
	if lo.Contains(manifest.Sections, "workspaces") {
		if err := s.importWorkspaces(ctx, zr, req, channelRemap, &resp); err != nil {
			return resp, err
		}
	}
	return resp, nil
}

// --- Analyze helpers ---

func (s *Service) analyzeChannels(ctx context.Context, zr *zip.Reader) ([]AnalysisItem, error) {
	channels, err := readArchiveSection[Channel](zr, "channels/")
	if err != nil {
		return nil, err
	}
	names := lo.Map(channels, func(c Channel, _ int) string { return c.Name })
	var existing []distchannel.Channel
	if len(names) > 0 {
		if err := s.cfg.Distribution.Channel.NewRetrieve().
			WhereNames(names...).Entries(&existing).Exec(ctx, nil); err != nil {
			return nil, errors.Wrap(err, "failed to retrieve existing channels")
		}
	}
	existingByName := lo.KeyBy(existing, func(c distchannel.Channel) string { return c.Name })
	items := make([]AnalysisItem, 0, len(channels))
	for _, ch := range channels {
		item := AnalysisItem{
			Type:       "channel",
			Name:       ch.Name,
			ArchiveKey: fmt.Sprintf("%d", ch.Key),
		}
		if ex, ok := existingByName[ch.Name]; ok {
			item.ExistingKey = fmt.Sprintf("%d", ex.Key())
			if ch.DataType == ex.DataType && ch.IsIndex == ex.IsIndex && ch.Virtual == ex.Virtual {
				item.Status = StatusIdentical
			} else {
				item.Status = StatusConflict
				item.Details = channelDiffDetails(ch, ex)
			}
		} else {
			item.Status = StatusNew
		}
		items = append(items, item)
	}
	return items, nil
}

func channelDiffDetails(archive Channel, existing distchannel.Channel) string {
	var diffs []string
	if archive.DataType != existing.DataType {
		diffs = append(diffs, fmt.Sprintf("data_type: %s vs %s", archive.DataType, existing.DataType))
	}
	if archive.IsIndex != existing.IsIndex {
		diffs = append(diffs, fmt.Sprintf("is_index: %v vs %v", archive.IsIndex, existing.IsIndex))
	}
	if archive.Virtual != existing.Virtual {
		diffs = append(diffs, fmt.Sprintf("virtual: %v vs %v", archive.Virtual, existing.Virtual))
	}
	return strings.Join(diffs, "; ")
}

func (s *Service) analyzeWorkspaces(ctx context.Context, zr *zip.Reader) ([]AnalysisItem, error) {
	workspaces, err := readArchiveSection[Workspace](zr, "workspaces/")
	if err != nil {
		return nil, err
	}
	names := lo.Map(workspaces, func(w Workspace, _ int) string { return w.Name })
	var existing []workspace.Workspace
	if len(names) > 0 {
		if err := s.cfg.Service.Workspace.NewRetrieve().
			WhereNames(names...).Entries(&existing).Exec(ctx, nil); err != nil {
			return nil, errors.Wrap(err, "failed to retrieve existing workspaces")
		}
	}
	existingByName := lo.KeyBy(existing, func(w workspace.Workspace) string { return w.Name })
	var items []AnalysisItem
	for _, ws := range workspaces {
		item := AnalysisItem{
			Type:       "workspace",
			Name:       ws.Name,
			ArchiveKey: ws.Key.String(),
		}
		if ex, ok := existingByName[ws.Name]; ok {
			item.ExistingKey = ex.Key.String()
			item.Status = StatusConflict
		} else {
			item.Status = StatusNew
		}
		items = append(items, item)

		childItems, err := s.analyzeWorkspaceChildren(ctx, zr, ws, existingByName)
		if err != nil {
			return nil, err
		}
		items = append(items, childItems...)
	}
	return items, nil
}

func (s *Service) analyzeWorkspaceChildren(
	ctx context.Context,
	zr *zip.Reader,
	ws Workspace,
	existingWS map[string]workspace.Workspace,
) ([]AnalysisItem, error) {
	wsKeyStr := ws.Key.String()
	var existingChildren map[string]uuid.UUID
	if ex, ok := existingWS[ws.Name]; ok {
		var err error
		existingChildren, err = s.getWorkspaceChildKeys(ctx, ex.Key)
		if err != nil {
			return nil, err
		}
	}
	var items []AnalysisItem
	prefix := fmt.Sprintf("workspaces/%s/", wsKeyStr)
	for _, f := range zr.File {
		if !strings.HasPrefix(f.Name, prefix) || !strings.HasSuffix(f.Name, ".json") {
			continue
		}
		parts := strings.Split(strings.TrimPrefix(f.Name, prefix), "/")
		if len(parts) != 2 {
			continue
		}
		childType := parts[0]
		name, err := readChildName(f)
		if err != nil {
			return nil, err
		}
		archiveKey := strings.TrimSuffix(parts[1], ".json")
		item := AnalysisItem{
			Type:       singularType(childType),
			Name:       name,
			ArchiveKey: archiveKey,
			ParentName: ws.Name,
		}
		childID := childMapKey(name, singularType(childType))
		if _, ok := existingChildren[childID]; ok {
			item.Status = StatusConflict
		} else {
			item.Status = StatusNew
		}
		items = append(items, item)
	}
	return items, nil
}

// getWorkspaceChildKeys returns a map of "name:type" → UUID key for all
// children of the given workspace. This is fetched once per workspace and
// reused for all child imports, avoiding N+1 queries.
func (s *Service) getWorkspaceChildKeys(ctx context.Context, wsKey uuid.UUID) (map[string]uuid.UUID, error) {
	var children []ontology.Resource
	if err := s.cfg.Distribution.Ontology.NewRetrieve().
		WhereIDs(workspace.OntologyID(wsKey)).
		TraverseTo(ontology.ChildrenTraverser).
		Entries(&children).
		Exec(ctx, nil); err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve children of workspace %s", wsKey)
	}
	result := make(map[string]uuid.UUID, len(children))
	for _, c := range children {
		key, err := uuid.Parse(c.ID.Key)
		if err != nil {
			continue
		}
		result[childMapKey(c.Name, string(c.ID.Type))] = key
	}
	return result, nil
}

func childMapKey(name, ontologyType string) string {
	return name + ":" + ontologyType
}

func singularType(plural string) string {
	switch plural {
	case "lineplots":
		return "lineplot"
	case "schematics":
		return "schematic"
	case "tables":
		return "table"
	case "logs":
		return "log"
	case "arcs":
		return "arc"
	}
	return plural
}

func readChildName(f *zip.File) (string, error) {
	rc, err := f.Open()
	if err != nil {
		return "", errors.Wrapf(err, "failed to open %s", f.Name)
	}
	defer rc.Close()
	var v struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(rc).Decode(&v); err != nil {
		return "", errors.Wrapf(err, "failed to decode %s", f.Name)
	}
	return v.Name, nil
}

func (s *Service) analyzeUsers(ctx context.Context, zr *zip.Reader) ([]AnalysisItem, error) {
	users, err := readArchiveSection[user.User](zr, "users/")
	if err != nil {
		return nil, err
	}
	usernames := lo.Map(users, func(u user.User, _ int) string { return u.Username })
	var existing []user.User
	if len(usernames) > 0 {
		if err := s.cfg.Service.User.NewRetrieve().
			WhereUsernames(usernames...).Entries(&existing).Exec(ctx, nil); err != nil {
			return nil, errors.Wrap(err, "failed to retrieve existing users")
		}
	}
	existingByName := lo.KeyBy(existing, func(u user.User) string { return u.Username })
	items := make([]AnalysisItem, 0, len(users))
	for _, u := range users {
		item := AnalysisItem{
			Type:       "user",
			Name:       u.Username,
			ArchiveKey: u.Key.String(),
		}
		if ex, ok := existingByName[u.Username]; ok {
			item.ExistingKey = ex.Key.String()
			item.Status = StatusIdentical
		} else {
			item.Status = StatusNew
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Service) analyzeDevices(ctx context.Context, zr *zip.Reader) ([]AnalysisItem, error) {
	devices, err := readArchiveSection[device.Device](zr, "devices/")
	if err != nil {
		return nil, err
	}
	names := lo.Map(devices, func(d device.Device, _ int) string { return d.Name })
	var existing []device.Device
	if len(names) > 0 {
		if err := s.cfg.Service.Device.NewRetrieve().
			WhereNames(names...).Entries(&existing).Exec(ctx, nil); err != nil {
			return nil, errors.Wrap(err, "failed to retrieve existing devices")
		}
	}
	existingByName := lo.KeyBy(existing, func(d device.Device) string { return d.Name })
	items := make([]AnalysisItem, 0, len(devices))
	for _, d := range devices {
		item := AnalysisItem{
			Type:       "device",
			Name:       d.Name,
			ArchiveKey: d.Key,
		}
		if ex, ok := existingByName[d.Name]; ok {
			item.ExistingKey = ex.Key
			item.Status = StatusConflict
		} else {
			item.Status = StatusNew
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Service) analyzeTasks(ctx context.Context, zr *zip.Reader) ([]AnalysisItem, error) {
	tasks, err := readArchiveSection[task.Task](zr, "tasks/")
	if err != nil {
		return nil, err
	}
	names := lo.Map(tasks, func(t task.Task, _ int) string { return t.Name })
	var existing []task.Task
	if len(names) > 0 {
		if err := s.cfg.Service.Task.NewRetrieve().
			WhereNames(names...).Entries(&existing).Exec(ctx, nil); err != nil {
			return nil, errors.Wrap(err, "failed to retrieve existing tasks")
		}
	}
	existingByName := lo.KeyBy(existing, func(t task.Task) string { return t.Name })
	items := make([]AnalysisItem, 0, len(tasks))
	for _, t := range tasks {
		item := AnalysisItem{
			Type:       "task",
			Name:       t.Name,
			ArchiveKey: fmt.Sprintf("%d", t.Key),
		}
		if ex, ok := existingByName[t.Name]; ok {
			item.ExistingKey = fmt.Sprintf("%d", ex.Key)
			item.Status = StatusConflict
		} else {
			item.Status = StatusNew
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Service) analyzeRanges(ctx context.Context, zr *zip.Reader) ([]AnalysisItem, error) {
	ranges, err := readArchiveSection[ranger.Range](zr, "ranges/")
	if err != nil {
		return nil, err
	}
	names := lo.Map(ranges, func(r ranger.Range, _ int) string { return r.Name })
	var existing []ranger.Range
	if len(names) > 0 {
		if err := s.cfg.Service.Ranger.NewRetrieve().
			WhereNames(names...).Entries(&existing).Exec(ctx, nil); err != nil {
			return nil, errors.Wrap(err, "failed to retrieve existing ranges")
		}
	}
	existingByName := lo.KeyBy(existing, func(r ranger.Range) string { return r.Name })
	items := make([]AnalysisItem, 0, len(ranges))
	for _, r := range ranges {
		item := AnalysisItem{
			Type:       "range",
			Name:       r.Name,
			ArchiveKey: r.Key.String(),
		}
		if ex, ok := existingByName[r.Name]; ok {
			item.ExistingKey = ex.Key.String()
			item.Status = StatusConflict
		} else {
			item.Status = StatusNew
		}
		items = append(items, item)
	}
	return items, nil
}

// --- Import helpers ---

func (s *Service) importChannels(
	ctx context.Context,
	zr *zip.Reader,
	req ImportRequest,
	remap map[distchannel.Key]distchannel.Key,
	resp *ImportResponse,
) error {
	channels, err := readArchiveSection[Channel](zr, "channels/")
	if err != nil {
		return err
	}
	for _, ch := range channels {
		archiveKey := fmt.Sprintf("%d", ch.Key)
		policy := req.PolicyFor(archiveKey)

		newCh := distchannel.Channel{
			Name:     ch.Name,
			DataType: ch.DataType,
			IsIndex:  ch.IsIndex,
			Virtual:  ch.Virtual,
		}
		var opts []distchannel.CreateOption
		switch policy {
		case PolicySkip:
			opts = append(opts, distchannel.RetrieveIfNameExists())
		case PolicyReplace:
			opts = append(opts, distchannel.RetrieveIfNameExists())
			opts = append(opts, distchannel.OverwriteIfNameExistsAndDifferentProperties())
		}
		if err := s.cfg.Service.Channel.Create(ctx, &newCh, opts...); err != nil {
			return errors.Wrapf(err, "failed to import channel %q", ch.Name)
		}
		newKey := newCh.Key()
		if newKey != ch.Key {
			remap[ch.Key] = newKey
		}
		if policy == PolicySkip && newKey == ch.Key {
			resp.Skipped++
		} else if newKey != ch.Key {
			resp.Imported++
		} else {
			resp.Replaced++
		}
	}
	return nil
}

func (s *Service) importRanges(
	ctx context.Context,
	zr *zip.Reader,
	req ImportRequest,
	resp *ImportResponse,
) error {
	ranges, err := readArchiveSection[ranger.Range](zr, "ranges/")
	if err != nil {
		return err
	}
	for _, r := range ranges {
		policy := req.PolicyFor(r.Key.String())
		var existing []ranger.Range
		if err := s.cfg.Service.Ranger.NewRetrieve().
			WhereNames(r.Name).Entries(&existing).Exec(ctx, nil); err != nil {
			return errors.Wrapf(err, "failed to check existing range %q", r.Name)
		}
		if len(existing) > 0 {
			if policy == PolicySkip {
				resp.Skipped++
				continue
			}
			// Replace: delete existing, then create new with same name.
			if err := s.cfg.Service.Ranger.NewWriter(nil).Delete(ctx, existing[0].Key); err != nil {
				return errors.Wrapf(err, "failed to delete existing range %q for replace", r.Name)
			}
			resp.Replaced++
		} else {
			resp.Imported++
		}
		r.Key = uuid.Nil
		if err := s.cfg.Service.Ranger.NewWriter(nil).Create(ctx, &r); err != nil {
			return errors.Wrapf(err, "failed to import range %q", r.Name)
		}
	}
	return nil
}

func (s *Service) importUsers(
	ctx context.Context,
	zr *zip.Reader,
	req ImportRequest,
	resp *ImportResponse,
) error {
	users, err := readArchiveSection[user.User](zr, "users/")
	if err != nil {
		return err
	}
	for _, u := range users {
		var existing []user.User
		if err := s.cfg.Service.User.NewRetrieve().
			WhereUsernames(u.Username).Entries(&existing).Exec(ctx, nil); err != nil {
			return errors.Wrapf(err, "failed to check existing user %q", u.Username)
		}
		if len(existing) > 0 {
			// Users are always skipped when they already exist — replacing a user
			// (credentials, key, etc.) is not a safe import operation.
			resp.Skipped++
			continue
		}
		u.Key = uuid.Nil
		if err := s.cfg.Service.User.NewWriter(nil).Create(ctx, &u); err != nil {
			return errors.Wrapf(err, "failed to import user %q", u.Username)
		}
		resp.Imported++
	}
	return nil
}

func (s *Service) importDevices(
	ctx context.Context,
	zr *zip.Reader,
	req ImportRequest,
	resp *ImportResponse,
) error {
	devices, err := readArchiveSection[device.Device](zr, "devices/")
	if err != nil {
		return err
	}
	for _, d := range devices {
		policy := req.PolicyFor(d.Key)
		var existing []device.Device
		if err := s.cfg.Service.Device.NewRetrieve().
			WhereNames(d.Name).Entries(&existing).Exec(ctx, nil); err != nil {
			return errors.Wrapf(err, "failed to check existing device %q", d.Name)
		}
		if len(existing) > 0 {
			if policy == PolicySkip {
				resp.Skipped++
				continue
			}
			// Replace: use existing device key so Create upserts.
			d.Key = existing[0].Key
			resp.Replaced++
		} else {
			resp.Imported++
		}
		if err := s.cfg.Service.Device.NewWriter(nil).Create(ctx, &d); err != nil {
			return errors.Wrapf(err, "failed to import device %q", d.Name)
		}
	}
	return nil
}

func (s *Service) importTasks(
	ctx context.Context,
	zr *zip.Reader,
	req ImportRequest,
	resp *ImportResponse,
) error {
	tasks, err := readArchiveSection[task.Task](zr, "tasks/")
	if err != nil {
		return err
	}
	for _, t := range tasks {
		policy := req.PolicyFor(fmt.Sprintf("%d", t.Key))
		var existing []task.Task
		if err := s.cfg.Service.Task.NewRetrieve().
			WhereNames(t.Name).Entries(&existing).Exec(ctx, nil); err != nil {
			return errors.Wrapf(err, "failed to check existing task %q", t.Name)
		}
		if len(existing) > 0 {
			if policy == PolicySkip {
				resp.Skipped++
				continue
			}
			// Replace: delete existing, then create new with same name.
			if err := s.cfg.Service.Task.NewWriter(nil).Delete(ctx, existing[0].Key, false); err != nil {
				return errors.Wrapf(err, "failed to delete existing task %q for replace", t.Name)
			}
			resp.Replaced++
		} else {
			resp.Imported++
		}
		t.Key = 0
		if err := s.cfg.Service.Task.NewWriter(nil).Create(ctx, &t); err != nil {
			return errors.Wrapf(err, "failed to import task %q", t.Name)
		}
	}
	return nil
}

func (s *Service) importWorkspaces(
	ctx context.Context,
	zr *zip.Reader,
	req ImportRequest,
	channelRemap map[distchannel.Key]distchannel.Key,
	resp *ImportResponse,
) error {
	workspaces, err := readArchiveSection[Workspace](zr, "workspaces/")
	if err != nil {
		return err
	}
	for _, ws := range workspaces {
		policy := req.PolicyFor(ws.Key.String())
		var existing []workspace.Workspace
		if err := s.cfg.Service.Workspace.NewRetrieve().
			WhereNames(ws.Name).Entries(&existing).Exec(ctx, nil); err != nil {
			return errors.Wrapf(err, "failed to check existing workspace %q", ws.Name)
		}
		var targetKey uuid.UUID
		if len(existing) > 0 {
			if policy == PolicySkip {
				resp.Skipped++
				targetKey = existing[0].Key
			} else {
				targetKey = existing[0].Key
				if err := s.cfg.Service.Workspace.NewWriter(nil).SetLayout(
					ctx, targetKey, string(ws.Layout),
				); err != nil {
					return errors.Wrapf(err, "failed to update workspace %q", ws.Name)
				}
				resp.Replaced++
			}
		} else {
			newWS := workspace.Workspace{
				Name:   ws.Name,
				Layout: string(ws.Layout),
				Author: ws.Author,
			}
			if err := s.cfg.Service.Workspace.NewWriter(nil).Create(ctx, &newWS); err != nil {
				return errors.Wrapf(err, "failed to import workspace %q", ws.Name)
			}
			targetKey = newWS.Key
			resp.Imported++
		}
		if err := s.importWorkspaceChildren(ctx, zr, ws, targetKey, req, channelRemap, resp); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) importWorkspaceChildren(
	ctx context.Context,
	zr *zip.Reader,
	ws Workspace,
	targetWSKey uuid.UUID,
	req ImportRequest,
	channelRemap map[distchannel.Key]distchannel.Key,
	resp *ImportResponse,
) error {
	prefix := fmt.Sprintf("workspaces/%s/", ws.Key.String())
	// Fetch all existing children once — avoids N+1 queries per child.
	existingChildren, err := s.getWorkspaceChildKeys(ctx, targetWSKey)
	if err != nil {
		return err
	}

	for _, f := range zr.File {
		if !strings.HasPrefix(f.Name, prefix) || !strings.HasSuffix(f.Name, ".json") {
			continue
		}
		parts := strings.Split(strings.TrimPrefix(f.Name, prefix), "/")
		if len(parts) != 2 {
			continue
		}
		childType := singularType(parts[0])
		archiveKey := strings.TrimSuffix(parts[1], ".json")
		policy := req.PolicyFor(archiveKey)

		switch childType {
		case "lineplot":
			if err := s.importDataViz(ctx, f, targetWSKey, childType, existingChildren,
				policy, channelRemap, resp, lineplot.OntologyType,
				func(name, data string) error {
					lp := lineplot.LinePlot{Name: name, Data: data}
					return s.cfg.Service.LinePlot.NewWriter(nil).Create(ctx, targetWSKey, &lp)
				},
				func(key uuid.UUID, data string) error {
					return s.cfg.Service.LinePlot.NewWriter(nil).SetData(ctx, key, data)
				},
			); err != nil {
				return err
			}
		case "schematic":
			if err := s.importSchematicViz(ctx, f, targetWSKey, existingChildren,
				policy, channelRemap, resp); err != nil {
				return err
			}
		case "table":
			if err := s.importDataViz(ctx, f, targetWSKey, childType, existingChildren,
				policy, channelRemap, resp, table.OntologyType,
				func(name, data string) error {
					t := table.Table{Name: name, Data: data}
					return s.cfg.Service.Table.NewWriter(nil).Create(ctx, targetWSKey, &t)
				},
				func(key uuid.UUID, data string) error {
					return s.cfg.Service.Table.NewWriter(nil).SetData(ctx, key, data)
				},
			); err != nil {
				return err
			}
		case "log":
			if err := s.importDataViz(ctx, f, targetWSKey, childType, existingChildren,
				policy, channelRemap, resp, log.OntologyType,
				func(name, data string) error {
					l := log.Log{Name: name, Data: data}
					return s.cfg.Service.Log.NewWriter(nil).Create(ctx, targetWSKey, &l)
				},
				func(key uuid.UUID, data string) error {
					return s.cfg.Service.Log.NewWriter(nil).SetData(ctx, key, data)
				},
			); err != nil {
				return err
			}
		}
	}
	return nil
}

// importDataViz is the shared logic for importing lineplots, tables, and logs
// (all stored as DataVisualization in the archive).
func (s *Service) importDataViz(
	_ context.Context,
	f *zip.File,
	_ uuid.UUID,
	childType string,
	existingChildren map[string]uuid.UUID,
	policy ConflictPolicy,
	channelRemap map[distchannel.Key]distchannel.Key,
	resp *ImportResponse,
	_ ontology.Type,
	create func(name, data string) error,
	setData func(key uuid.UUID, data string) error,
) error {
	dv, err := readZipFile[DataVisualization](f)
	if err != nil {
		return err
	}
	data, err := remapChannelKeys(dv.Data, channelRemap)
	if err != nil {
		return errors.Wrapf(err, "failed to remap channel keys in %s %q", childType, dv.Name)
	}
	mapKey := childMapKey(dv.Name, childType)
	if existingKey, exists := existingChildren[mapKey]; exists {
		if policy == PolicySkip {
			resp.Skipped++
			return nil
		}
		if err := setData(existingKey, string(data)); err != nil {
			return errors.Wrapf(err, "failed to update %s %q", childType, dv.Name)
		}
		resp.Replaced++
		return nil
	}
	if err := create(dv.Name, string(data)); err != nil {
		return errors.Wrapf(err, "failed to import %s %q", childType, dv.Name)
	}
	resp.Imported++
	return nil
}

// importSchematicViz handles schematic import separately because schematics have
// the extra Snapshot field (not a DataVisualization).
func (s *Service) importSchematicViz(
	ctx context.Context,
	f *zip.File,
	wsKey uuid.UUID,
	existingChildren map[string]uuid.UUID,
	policy ConflictPolicy,
	channelRemap map[distchannel.Key]distchannel.Key,
	resp *ImportResponse,
) error {
	sch, err := readZipFile[Schematic](f)
	if err != nil {
		return err
	}
	data, err := remapChannelKeys(sch.Data, channelRemap)
	if err != nil {
		return errors.Wrapf(err, "failed to remap channel keys in schematic %q", sch.Name)
	}
	mapKey := childMapKey(sch.Name, "schematic")
	if existingKey, exists := existingChildren[mapKey]; exists {
		if policy == PolicySkip {
			resp.Skipped++
			return nil
		}
		if err := s.cfg.Service.Schematic.NewWriter(nil).SetData(ctx, existingKey, string(data)); err != nil {
			return errors.Wrapf(err, "failed to update schematic %q", sch.Name)
		}
		resp.Replaced++
		return nil
	}
	newSch := schematic.Schematic{Name: sch.Name, Data: string(data), Snapshot: sch.Snapshot}
	if err := s.cfg.Service.Schematic.NewWriter(nil).Create(ctx, wsKey, &newSch); err != nil {
		return errors.Wrapf(err, "failed to import schematic %q", sch.Name)
	}
	resp.Imported++
	return nil
}

// --- Archive reading helpers ---

// readArchiveJSON reads and decodes a single JSON file from the archive.
func readArchiveJSON[T any](zr *zip.Reader, path string) (T, error) {
	var v T
	for _, f := range zr.File {
		if f.Name == path {
			return readZipFile[T](f)
		}
	}
	return v, errors.Newf("file not found in archive: %s", path)
}

// readArchiveSection reads all JSON files in a directory prefix from the archive.
func readArchiveSection[T any](zr *zip.Reader, prefix string) ([]T, error) {
	var items []T
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, prefix) && strings.HasSuffix(f.Name, ".json") {
			remainder := strings.TrimPrefix(f.Name, prefix)
			if strings.Contains(remainder, "/") {
				continue
			}
			v, err := readZipFile[T](f)
			if err != nil {
				return nil, err
			}
			items = append(items, v)
		}
	}
	return items, nil
}

// readZipFile decodes a single ZIP file entry as JSON.
func readZipFile[T any](f *zip.File) (T, error) {
	var v T
	rc, err := f.Open()
	if err != nil {
		return v, errors.Wrapf(err, "failed to open %s", f.Name)
	}
	defer rc.Close()
	if err := json.NewDecoder(rc).Decode(&v); err != nil {
		return v, errors.Wrapf(err, "failed to decode %s", f.Name)
	}
	return v, nil
}
