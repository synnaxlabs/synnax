// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { manifest } from "@/index";

/**
 * The production manifest: one entry per docs video, keyed by the id the docs
 * site's Video component uses. `pnpm batch` produces the light/dark pair for
 * every entry; `pnpm upload` pushes them to the docs CDN.
 */
export default manifest.define([
  {
    id: "console/ui-overview/open-toolbars",
    script: "scripts/ui-overview-open-toolbars.ts",
  },
  {
    id: "console/ui-overview/range-toolbar",
    script: "scripts/ui-overview-range-toolbar.ts",
  },
  {
    id: "console/ui-overview/palette-search",
    script: "scripts/ui-overview-palette-search.ts",
  },
  {
    id: "console/ui-overview/palette-command",
    script: "scripts/ui-overview-palette-command.ts",
  },
  {
    id: "console/ui-overview/visualization-toolbar",
    script: "scripts/ui-overview-visualization-toolbar.ts",
  },
  { id: "console/line-plots/toolbar", script: "scripts/line-plot-toolbar.ts" },
  { id: "console/line-plots/data-tab", script: "scripts/line-plot-data.ts" },
  { id: "console/line-plots/lines-tab", script: "scripts/line-plot-lines.ts" },
  { id: "console/line-plots/axes-tab", script: "scripts/line-plot-axes.ts" },
  {
    id: "console/line-plots/properties-tab",
    script: "scripts/line-plot-properties.ts",
  },
  { id: "console/line-plots/zoom", script: "scripts/line-plot-zoom.ts" },
  { id: "console/line-plots/pan", script: "scripts/line-plot-pan.ts" },
  { id: "console/line-plots/select", script: "scripts/line-plot-select.ts" },
  { id: "console/line-plots/slope", script: "scripts/line-plot-slope.ts" },
  {
    id: "console/ui-overview/multiple-tabs",
    script: "scripts/ui-overview-multiple-tabs.ts",
  },
  {
    id: "console/ui-overview/close-toolbars",
    script: "scripts/ui-overview-close-toolbars.ts",
  },
  {
    id: "console/ui-overview/documentation",
    script: "scripts/ui-overview-documentation.ts",
  },
  { id: "console/tables/create", script: "scripts/table-create.ts" },
  { id: "console/logs/example", script: "scripts/log-example.ts" },
  { id: "console/schematics/create", script: "scripts/schematic-create.ts" },
  {
    id: "console/schematics/connections",
    script: "scripts/schematic-connections.ts",
  },
  {
    id: "console/schematics/align-items",
    script: "scripts/schematic-align-items.ts",
  },
  {
    id: "console/schematics/change-color",
    script: "scripts/schematic-change-color.ts",
  },
  { id: "console/schematics/value", script: "scripts/schematic-value.ts" },
  {
    id: "console/schematics/symbol-create-group",
    script: "scripts/schematic-symbol-create-group.ts",
  },
  {
    id: "console/schematics/symbol-import-svg",
    script: "scripts/schematic-symbol-import-svg.ts",
  },
  { id: "console/schematics/download", script: "scripts/schematic-download.ts" },
  { id: "console/schematics/upload", script: "scripts/schematic-upload.ts" },
  { id: "console/schematics/snapshot", script: "scripts/schematic-snapshot.ts" },
  { id: "console/schematics/valve", script: "scripts/schematic-valve.ts" },
  // Both shots put the Core's address on screen, so they run on the port the
  // docs quote.
  {
    id: "console/ui-overview/cluster-toolbar",
    script: "scripts/ui-overview-cluster-toolbar.ts",
    port: 9090,
  },
  {
    id: "console/clusters/connect",
    script: "scripts/clusters-connect.ts",
    port: 9090,
  },
  { id: "console/channels/create", script: "scripts/channels-create.ts" },
  { id: "console/channels/alias", script: "scripts/channels-alias.ts" },
  {
    id: "console/calculated-channels/create",
    script: "scripts/calculated-channels-create.ts",
  },
  {
    id: "console/calculated-channels/edit",
    script: "scripts/calculated-channels-edit.ts",
  },
  { id: "console/ranges/toolbar-create", script: "scripts/ranges-toolbar-create.ts" },
  { id: "console/ranges/palette-create", script: "scripts/ranges-palette-create.ts" },
  { id: "console/ranges/resources", script: "scripts/ranges-resources.ts" },
  { id: "console/ranges/palette", script: "scripts/ranges-palette.ts" },
  { id: "console/ranges/create-child", script: "scripts/ranges-create-child.ts" },
  { id: "console/ranges/add-meta-data", script: "scripts/ranges-add-meta-data.ts" },
  { id: "console/ranges/add-label", script: "scripts/ranges-add-label.ts" },
  { id: "console/users/register", script: "scripts/users-register.ts" },
  {
    id: "console/users/modal-change-role",
    script: "scripts/users-modal-change-role.ts",
  },
  {
    id: "console/users/drag-change-role",
    script: "scripts/users-drag-change-role.ts",
  },
  { id: "console/ranges/plot-create", script: "scripts/ranges-plot-create.ts" },
  {
    id: "control/arc/get-started/create-automation",
    script: "scripts/arc-create-automation.ts",
  },
  {
    id: "device-drivers/task/command-palette",
    script: "scripts/task-command-palette.ts",
  },
  {
    id: "device-drivers/task/layout-selector",
    script: "scripts/task-layout-selector.ts",
  },
]);
