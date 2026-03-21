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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/samber/lo"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
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
	userRemap := make(map[uuid.UUID]uuid.UUID)

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
		if err := s.importUsers(ctx, zr, req, userRemap, &resp); err != nil {
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
		if err := s.importWorkspaces(ctx, zr, req, channelRemap, userRemap, &resp); err != nil {
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

	// Build a set of all channel keys present in the archive so we can detect
	// data channels whose index channel was not included in the export.
	archiveKeys := make(map[distchannel.Key]bool, len(channels))
	for _, ch := range channels {
		archiveKeys[ch.Key] = true
	}

	items := make([]AnalysisItem, 0, len(channels))
	for _, ch := range channels {
		item := AnalysisItem{
			Type:       "channel",
			Name:       ch.Name,
			ArchiveKey: fmt.Sprintf("%d", ch.Key),
			DataType:   ch.DataType,
			ParentName: ch.Group,
		}
		// A non-virtual, non-index channel requires its index channel. If the
		// index is not in the archive AND doesn't already exist on this system,
		// the channel cannot be imported.
		if !ch.IsIndex && !ch.Virtual && ch.Index != 0 {
			_, indexInArchive := archiveKeys[ch.Index]
			_, indexExists := existingByName[ch.Name]
			// Check if the index channel exists on the system by key
			if !indexInArchive {
				var indexChannels []distchannel.Channel
				if err := s.cfg.Distribution.Channel.NewRetrieve().
					WhereKeys(ch.Index).Entries(&indexChannels).Exec(ctx, nil); err == nil {
					indexExists = len(indexChannels) > 0
				}
			}
			if !indexInArchive && !indexExists {
				item.Status = StatusNew
				item.Disabled = true
				item.Details = "index channel not included in archive"
				items = append(items, item)
				continue
			}
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
			keyMatch := ex.Key == ws.Key
			layoutMatch := compactJSON(ex.Layout) == compactJSON(string(ws.Layout))
			if keyMatch && layoutMatch {
				item.Status = StatusIdentical
			} else {
				item.Status = StatusConflict
				var diffs []string
				if !keyMatch {
					diffs = append(diffs, "different key")
				}
				if !layoutMatch {
					diffs = append(diffs, "layout differs")
				}
				item.Details = strings.Join(diffs, "; ")
			}
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
		childType := singularType(parts[0])
		archiveChild, err := readZipFile[DataVisualization](f)
		if err != nil {
			return nil, err
		}
		archiveKey := strings.TrimSuffix(parts[1], ".json")
		item := AnalysisItem{
			Type:       childType,
			Name:       archiveChild.Name,
			ArchiveKey: archiveKey,
			ParentName: ws.Name,
		}
		childID := childMapKey(archiveChild.Name, childType)
		if existingKey, ok := existingChildren[childID]; ok {
			existingData, err := s.getChildData(ctx, childType, existingKey)
			if err != nil {
				item.Status = StatusConflict
				item.Details = fmt.Sprintf("could not retrieve existing: %v", err)
			} else if compactJSON(string(archiveChild.Data)) == compactJSON(existingData) {
				item.Status = StatusIdentical
			} else {
				item.Status = StatusConflict
				item.Details = "data differs"
			}
			item.ExistingKey = existingKey.String()
		} else {
			item.Status = StatusNew
		}
		items = append(items, item)
	}
	return items, nil
}

// getChildData retrieves the Data field of a workspace child by type and key.
func (s *Service) getChildData(ctx context.Context, childType string, key uuid.UUID) (string, error) {
	switch childType {
	case "lineplot":
		var entries []lineplot.LinePlot
		if err := s.cfg.Service.LinePlot.NewRetrieve().WhereKeys(key).Entries(&entries).Exec(ctx, nil); err != nil || len(entries) == 0 {
			return "", errors.New("not found")
		}
		return entries[0].Data, nil
	case "schematic":
		var entries []schematic.Schematic
		if err := s.cfg.Service.Schematic.NewRetrieve().WhereKeys(key).Entries(&entries).Exec(ctx, nil); err != nil || len(entries) == 0 {
			return "", errors.New("not found")
		}
		return entries[0].Data, nil
	case "table":
		var entries []table.Table
		if err := s.cfg.Service.Table.NewRetrieve().WhereKeys(key).Entries(&entries).Exec(ctx, nil); err != nil || len(entries) == 0 {
			return "", errors.New("not found")
		}
		return entries[0].Data, nil
	case "log":
		var entries []log.Log
		if err := s.cfg.Service.Log.NewRetrieve().WhereKeys(key).Entries(&entries).Exec(ctx, nil); err != nil || len(entries) == 0 {
			return "", errors.New("not found")
		}
		return entries[0].Data, nil
	}
	return "", errors.Newf("unknown child type %q", childType)
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
	// Retrieve all existing devices that match by name OR location.
	names := lo.Map(devices, func(d device.Device, _ int) string { return d.Name })
	locations := lo.FilterMap(devices, func(d device.Device, _ int) (string, bool) {
		return d.Location, d.Location != ""
	})
	var byName, byLocation []device.Device
	if len(names) > 0 {
		if err := s.cfg.Service.Device.NewRetrieve().
			WhereNames(names...).Entries(&byName).Exec(ctx, nil); err != nil {
			return nil, errors.Wrap(err, "failed to retrieve existing devices by name")
		}
	}
	if len(locations) > 0 {
		if err := s.cfg.Service.Device.NewRetrieve().
			WhereLocations(locations...).Entries(&byLocation).Exec(ctx, nil); err != nil {
			return nil, errors.Wrap(err, "failed to retrieve existing devices by location")
		}
	}
	existingByName := lo.KeyBy(byName, func(d device.Device) string { return d.Name })
	existingByLocation := lo.KeyBy(byLocation, func(d device.Device) string { return d.Location })
	items := make([]AnalysisItem, 0, len(devices))
	for _, d := range devices {
		item := AnalysisItem{
			Type:       "device",
			Name:       d.Name,
			ArchiveKey: d.Key,
		}
		// Match by name first, then by location.
		ex, nameMatch := existingByName[d.Name]
		if !nameMatch && d.Location != "" {
			if locMatch, ok := existingByLocation[d.Location]; ok {
				ex = locMatch
				nameMatch = true
			}
		}
		if nameMatch {
			item.ExistingKey = ex.Key
			var diffs []string
			if ex.Name != d.Name {
				diffs = append(diffs, fmt.Sprintf("name: %s vs %s", ex.Name, d.Name))
			}
			if ex.Make != d.Make {
				diffs = append(diffs, fmt.Sprintf("make: %s vs %s", ex.Make, d.Make))
			}
			if ex.Model != d.Model {
				diffs = append(diffs, fmt.Sprintf("model: %s vs %s", ex.Model, d.Model))
			}
			if ex.Location != d.Location {
				diffs = append(diffs, fmt.Sprintf("location: %s vs %s", ex.Location, d.Location))
			}
			// Compare only the "connection" key from properties — the "read"
			// and "write" keys contain volatile channel mappings set by the
			// driver when tasks are configured.
			if !jsonEqual(ex.Properties["connection"], d.Properties["connection"]) {
				diffs = append(diffs, "connection config differs")
			}
			if len(diffs) == 0 {
				item.Status = StatusIdentical
			} else {
				item.Status = StatusConflict
				item.Details = strings.Join(diffs, "; ")
			}
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
			if ex.Type == t.Type && jsonEqual(ex.Config, t.Config) {
				item.Status = StatusIdentical
			} else {
				item.Status = StatusConflict
				var diffs []string
				if ex.Type != t.Type {
					diffs = append(diffs, fmt.Sprintf("type: %s vs %s", ex.Type, t.Type))
				}
				if !jsonEqual(ex.Config, t.Config) {
					diffs = append(diffs, "config differs")
				}
				item.Details = strings.Join(diffs, "; ")
			}
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
			if ex.TimeRange == r.TimeRange && ex.Color == r.Color {
				item.Status = StatusIdentical
			} else {
				item.Status = StatusConflict
				var diffs []string
				if ex.TimeRange != r.TimeRange {
					diffs = append(diffs, "time range differs")
				}
				if ex.Color != r.Color {
					diffs = append(diffs, "color differs")
				}
				item.Details = strings.Join(diffs, "; ")
			}
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
	// Sort: index channels first so their remapped keys are available when
	// we process data channels that reference them.
	indexChannels := lo.Filter(channels, func(c Channel, _ int) bool { return c.IsIndex })
	dataChannels := lo.Filter(channels, func(c Channel, _ int) bool { return !c.IsIndex })
	ordered := append(indexChannels, dataChannels...)

	// Build a set of archive keys for index resolution.
	archiveKeys := make(map[distchannel.Key]bool, len(ordered))
	for _, ch := range ordered {
		archiveKeys[ch.Key] = true
	}

	for _, ch := range ordered {
		archiveKey := fmt.Sprintf("%d", ch.Key)
		policy := req.PolicyFor(archiveKey)

		// Skip non-virtual data channels whose index channel is not available
		// (neither in the archive nor already on this system via remap).
		if !ch.IsIndex && !ch.Virtual && ch.Index != 0 {
			_, indexInArchive := archiveKeys[ch.Index]
			_, indexRemapped := remap[ch.Index]
			if !indexInArchive && !indexRemapped {
				// Check if the index channel already exists on this system.
				var indexChannels []distchannel.Channel
				if err := s.cfg.Distribution.Channel.NewRetrieve().
					WhereKeys(ch.Index).Entries(&indexChannels).Exec(ctx, nil); err != nil || len(indexChannels) == 0 {
					resp.Skipped++
					continue
				}
			}
		}

		newCh := distchannel.Channel{
			Name:     ch.Name,
			DataType: ch.DataType,
			IsIndex:  ch.IsIndex,
			Virtual:  ch.Virtual,
		}
		// For non-index channels, resolve their index channel key through
		// the remap table, then extract the LocalKey portion.
		if !ch.IsIndex && ch.Index != 0 {
			resolvedIndex := ch.Index
			if remapped, ok := remap[ch.Index]; ok {
				resolvedIndex = remapped
			}
			newCh.LocalIndex = resolvedIndex.LocalKey()
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
		if newKey == ch.Key && policy == PolicySkip {
			// Channel already existed with same key — it's identical, not skipped.
			resp.Identical++
		} else if newKey != ch.Key {
			resp.Imported++
		} else {
			resp.Replaced++
		}

		// Move channel to its custom group if one was specified in the archive.
		if ch.Group != "" {
			if err := s.moveChannelToGroup(ctx, newKey, ch.Group); err != nil {
				return errors.Wrapf(err, "failed to assign channel %q to group %q", ch.Name, ch.Group)
			}
		}
	}
	return nil
}

// moveChannelToGroup creates (or retrieves) a named group under the default
// "Channels" root group and moves the channel's ontology relationship to it.
func (s *Service) moveChannelToGroup(ctx context.Context, channelKey distchannel.Key, groupName string) error {
	channelsGroup := s.cfg.Distribution.Channel.Group()
	g, err := s.cfg.Distribution.Group.CreateOrRetrieve(ctx, groupName, group.OntologyID(channelsGroup.Key))
	if err != nil {
		return err
	}
	otgWriter := s.cfg.Distribution.Ontology.NewWriter(nil)
	channelID := distchannel.OntologyID(channelKey)
	// Remove from the default "Channels" group.
	if err := otgWriter.DeleteRelationship(
		ctx,
		group.OntologyID(channelsGroup.Key),
		ontology.RelationshipTypeParentOf,
		channelID,
	); err != nil {
		return err
	}
	// Add to the custom group.
	return otgWriter.DefineRelationship(
		ctx,
		group.OntologyID(g.Key),
		ontology.RelationshipTypeParentOf,
		channelID,
	)
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
				if existing[0].TimeRange == r.TimeRange && existing[0].Color == r.Color {
					resp.Identical++
				} else {
					resp.Skipped++
				}
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
	userRemap map[uuid.UUID]uuid.UUID,
	resp *ImportResponse,
) error {
	users, err := readArchiveSection[user.User](zr, "users/")
	if err != nil {
		return err
	}
	for _, u := range users {
		archiveKey := u.Key
		var existing []user.User
		if err := s.cfg.Service.User.NewRetrieve().
			WhereUsernames(u.Username).Entries(&existing).Exec(ctx, nil); err != nil {
			return errors.Wrapf(err, "failed to check existing user %q", u.Username)
		}
		if len(existing) > 0 {
			// Users are always kept as-is when they already exist.
			if archiveKey != existing[0].Key {
				userRemap[archiveKey] = existing[0].Key
			}
			resp.Identical++
			continue
		}
		u.Key = uuid.Nil
		if err := s.cfg.Service.User.NewWriter(nil).Create(ctx, &u); err != nil {
			return errors.Wrapf(err, "failed to import user %q", u.Username)
		}
		if u.Key != archiveKey {
			userRemap[archiveKey] = u.Key
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
				ex := existing[0]
				if ex.Make == d.Make && ex.Model == d.Model &&
					ex.Location == d.Location && ex.Name == d.Name &&
					jsonEqual(ex.Properties["connection"], d.Properties["connection"]) {
					resp.Identical++
				} else {
					resp.Skipped++
				}
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
				if existing[0].Type == t.Type && jsonEqual(existing[0].Config, t.Config) {
					resp.Identical++
				} else {
					resp.Skipped++
				}
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
		t.Key = task.NewKey(s.cfg.Service.Rack.EmbeddedKey, 0)
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
	userRemap map[uuid.UUID]uuid.UUID,
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
		// Resolve the workspace author through the user remap table.
		// If the author isn't in the remap and doesn't exist on this system,
		// fall back to the first available user.
		author := ws.Author
		if remapped, ok := userRemap[author]; ok {
			author = remapped
		} else {
			var authorUsers []user.User
			if err := s.cfg.Service.User.NewRetrieve().
				WhereKeys(author).Entries(&authorUsers).Exec(ctx, nil); err != nil || len(authorUsers) == 0 {
				// Author doesn't exist — use the first user on this system.
				var allUsers []user.User
				if err := s.cfg.Service.User.NewRetrieve().
					Entries(&allUsers).Exec(ctx, nil); err == nil && len(allUsers) > 0 {
					author = allUsers[0].Key
				}
			}
		}
		var targetKey uuid.UUID
		if len(existing) > 0 {
			if policy == PolicySkip {
				if compactJSON(existing[0].Layout) == compactJSON(string(ws.Layout)) {
					resp.Identical++
				} else {
					resp.Skipped++
				}
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
				Author: author,
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
			existingData, _ := s.getChildData(ctx, childType, existingKey)
			if compactJSON(string(data)) == compactJSON(existingData) {
				resp.Identical++
			} else {
				resp.Skipped++
			}
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

// jsonEqual compares two values by marshaling them to compact JSON. This handles
// map[string]any comparisons where reflect.DeepEqual may fail due to numeric
// type differences (float64 vs int).
func jsonEqual(a, b any) bool {
	aj, err1 := json.Marshal(a)
	bj, err2 := json.Marshal(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return compactJSON(string(aj)) == compactJSON(string(bj))
}

// compactJSON removes formatting differences (whitespace, indentation) from a
// JSON string so two semantically equal JSON values compare as equal.
func compactJSON(s string) string {
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(s)); err != nil {
		return s
	}
	return buf.String()
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
