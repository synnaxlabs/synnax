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
    id: "console/ui-overview/palette-search",
    script: "scripts/ui-overview-palette-search.ts",
  },
  { id: "console/line-plots/toolbar", script: "scripts/line-plot-toolbar.ts" },
  { id: "console/line-plots/slope", script: "scripts/line-plot-slope.ts" },
  {
    id: "console/ui-overview/documentation",
    script: "scripts/ui-overview-documentation.ts",
  },
  { id: "console/logs/example", script: "scripts/log-example.ts" },
  {
    id: "console/schematics/connections",
    script: "scripts/schematic-connections.ts",
  },
  {
    id: "console/schematics/align-items",
    script: "scripts/schematic-align-items.ts",
  },
  {
    id: "console/schematics/edit-several-elements",
    script: "scripts/schematic-edit-several-elements.ts",
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
  { id: "console/schematics/snapshot", script: "scripts/schematic-snapshot.ts" },
  { id: "console/schematics/valve", script: "scripts/schematic-valve.ts" },
  // The shot puts the Core's address on screen, so it runs on the port the docs quote.
  {
    id: "console/clusters/connect",
    script: "scripts/clusters-connect.ts",
    port: 9090,
  },
  // A desktop-sized viewport, so the Console reads at its normal scale.
  {
    id: "client/channels/create",
    script: "scripts/channels-create.ts",
    width: 1600,
    height: 900,
  },
  {
    id: "client/channels/toolbar",
    script: "scripts/channels-toolbar.ts",
    width: 1600,
    height: 900,
  },
  {
    id: "client/channels/rename",
    script: "scripts/channels-rename.ts",
    width: 1600,
    height: 900,
  },
  {
    id: "client/channels/delete",
    script: "scripts/channels-delete.ts",
    width: 1600,
    height: 900,
  },
  { id: "console/channels/alias", script: "scripts/channels-alias.ts" },
  {
    id: "console/calculated-channels/create-and-edit",
    script: "scripts/calculated-channels-create-and-edit.ts",
  },
  { id: "client/ranges/create", script: "scripts/ranges-create.ts" },
  { id: "client/ranges/create-child", script: "scripts/ranges-create-child.ts" },
  { id: "client/ranges/set-metadata", script: "scripts/ranges-set-metadata.ts" },
  { id: "client/ranges/labels", script: "scripts/ranges-labels.ts" },
  { id: "console/workspaces/create", script: "scripts/projects-create.ts" },
  { id: "console/workspaces/load", script: "scripts/projects-load.ts" },
  { id: "console/users/register", script: "scripts/users-register.ts" },
  {
    id: "console/users/modal-change-role",
    script: "scripts/users-modal-change-role.ts",
  },
  {
    id: "control/arc/get-started/create-automation",
    script: "scripts/arc-create-automation.ts",
  },
  {
    id: "control/authority/hand-off",
    script: "scripts/control-authority-hand-off.ts",
  },
  {
    id: "control/manual/absolute-control",
    script: "scripts/control-manual-absolute-control.ts",
  },
  { id: "control/arc/overview", script: "scripts/control-arc-overview.ts" },
  {
    id: "control/arc/start-from-schematic/start-button",
    script: "scripts/control-arc-start-button.ts",
  },
  {
    id: "control/arc/graph-editor/build-alarm",
    script: "scripts/control-arc-build-alarm.ts",
  },
  {
    id: "device-drivers/task/command-palette",
    script: "scripts/task-command-palette.ts",
  },
  { id: "console/line-plots/create", script: "scripts/line-plot-create.ts" },
  { id: "console/logs/create", script: "scripts/log-create.ts" },
  { id: "console/schematics/create", script: "scripts/schematic-create.ts" },
  { id: "console/tables/create", script: "scripts/table-create.ts" },
  {
    id: "device-drivers/task/layout-selector",
    script: "scripts/task-layout-selector.ts",
  },
  // Hardware shots run against the simulated Driver in fixtures/hardware.ts.
  { id: "device-drivers/task/task-basic", script: "scripts/tasks-start.ts" },
  { id: "hardware/devices/toolbar", script: "scripts/devices-toolbar.ts" },
  // The custom devices tutorial runs against fixtures/arduino.ts.
  {
    id: "client/resources/build-device-driver/line-plot",
    script: "scripts/custom-devices-line-plot.ts",
  },
  {
    id: "client/resources/build-device-driver/schematic",
    script: "scripts/custom-devices-schematic.ts",
  },
  {
    id: "client/resources/build-device-driver/console-setup",
    script: "scripts/custom-devices-console-setup.ts",
  },
  {
    id: "releases/0-57-0/panels",
    script: "scripts/release-0-57-0-panels.ts",
    themed: false,
  },
]);
