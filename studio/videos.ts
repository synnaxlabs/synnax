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
 * A page or release hero: one 1280 x 800 Console still that the docs `Hero` component
 * frames. Batch writes its regions where `Hero` finds them by id.
 */
const hero = (id: string, script: string): manifest.Entry => ({
  id,
  script,
  dsf: 3,
  stills: { regions: `../site/docs/src/components/hero/stills/${id}.json` },
});

/**
 * The production manifest: one entry per docs video or set of stills, keyed by the id
 * the docs site requests. `pnpm batch` produces the light and dark media for every
 * entry; `pnpm upload` pushes them to the docs CDN.
 */
export default manifest.define([
  {
    id: "console/ui-overview/tour",
    script: "scripts/still-ui-overview.ts",
    dsf: 3,
    stills: { regions: "../site/docs/src/components/console/tour/regions.json" },
  },
  hero("console/line-plots/hero", "scripts/still-line-plot-hero.ts"),
  hero("console/schematics/hero", "scripts/still-schematic-hero.ts"),
  hero("console/tables/hero", "scripts/still-table-hero.ts"),
  hero("console/logs/hero", "scripts/still-log-hero.ts"),
  hero("console/visualize/hero", "scripts/still-visualize-hero.ts"),
  hero("console/ranges/hero", "scripts/still-range-hero.ts"),
  hero("control/arc/hero", "scripts/still-arc-hero.ts"),
  hero("control/arc/graph-editor/hero", "scripts/still-arc-graph-hero.ts"),
  hero("console/alarms/hero", "scripts/still-alarm-hero.ts"),
  hero("hardware/hero", "scripts/still-hardware-hero.ts"),
  hero("console/projects/hero", "scripts/still-project-hero.ts"),
  hero("releases/0-60-0/hero", "scripts/release-0-60-hero.ts"),
  hero("releases/0-60-0/read-back", "scripts/release-0-60-read-back.ts"),
  hero("releases/0-60-0/connect-device", "scripts/release-0-60-connect-device.ts"),
  hero("releases/0-60-0/panel-icons", "scripts/release-0-60-panel-icons.ts"),
  { id: "console/line-plots/toolbar", script: "scripts/line-plot-toolbar.ts" },
  { id: "console/line-plots/slope", script: "scripts/line-plot-slope.ts" },
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
  { id: "client/channels/create", script: "scripts/channels-create.ts" },
  {
    id: "client/channels/create-virtual",
    script: "scripts/channels-create-virtual.ts",
  },
  { id: "client/channels/find", script: "scripts/channels-find.ts" },
  { id: "client/channels/toolbar", script: "scripts/channels-toolbar.ts" },
  { id: "client/channels/rename", script: "scripts/channels-rename.ts" },
  { id: "client/channels/delete", script: "scripts/channels-delete.ts" },
  { id: "console/channels/alias", script: "scripts/channels-alias.ts" },
  {
    id: "client/calculated-channels/create",
    script: "scripts/calculated-channels-create.ts",
  },
  {
    id: "client/calculated-channels/edit",
    script: "scripts/calculated-channels-edit.ts",
  },
  { id: "client/ranges/create", script: "scripts/ranges-create.ts" },
  { id: "client/ranges/create-child", script: "scripts/ranges-create-child.ts" },
  { id: "client/ranges/set-metadata", script: "scripts/ranges-set-metadata.ts" },
  { id: "client/ranges/labels", script: "scripts/ranges-labels.ts" },
  { id: "console/ranges/active", script: "scripts/ranges-active.ts" },
  { id: "console/ranges/stages", script: "scripts/ranges-stages.ts" },
  { id: "console/ranges/export", script: "scripts/ranges-export.ts" },
  {
    id: "console/ranges/snapshot-task",
    script: "scripts/ranges-snapshot-task.ts",
  },
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
    id: "control/arc/get-started/first-stages",
    script: "scripts/arc-first-stages.ts",
  },
  { id: "control/arc/get-started/abort", script: "scripts/arc-abort.ts" },
  {
    id: "get-started/sample-project/open",
    script: "scripts/sample-project-open.ts",
  },
  {
    id: "get-started/sample-project/run-test",
    script: "scripts/sample-project-run-test.ts",
  },
  {
    id: "get-started/sample-project/take-control",
    script: "scripts/sample-project-take-control.ts",
  },
  {
    id: "get-started/sample-project/abort",
    script: "scripts/sample-project-abort.ts",
  },
  {
    id: "get-started/install-desktop",
    script: "scripts/still-install-desktop.ts",
    stills: {
      regions: "../site/docs/src/components/media/stills/install-desktop.json",
    },
  },
  { id: "get-started/first-run", script: "scripts/first-run.ts" },
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
  { id: "console/line-plots/create", script: "scripts/line-plot-create.ts" },
  { id: "console/logs/create", script: "scripts/log-create.ts" },
  { id: "console/schematics/create", script: "scripts/schematic-create.ts" },
  { id: "console/tables/create", script: "scripts/table-create.ts" },
  { id: "console/tables/value-cell", script: "scripts/table-value-cell.ts" },
  {
    id: "console/schematics/control-tab",
    script: "scripts/schematic-control-tab.ts",
  },
  { id: "console/panels/create", script: "scripts/panels-create.ts" },
  {
    id: "console/panels/rename-and-delete",
    script: "scripts/panels-rename-and-delete.ts",
  },
  { id: "console/panels/open-tab", script: "scripts/panels-open-tab.ts" },
  { id: "console/panels/split", script: "scripts/panels-split.ts" },
  { id: "console/panels/move", script: "scripts/panels-move.ts" },
  { id: "console/panels/focus", script: "scripts/panels-focus.ts" },
  // Hardware shots run against the simulated Driver in fixtures/hardware.ts.
  { id: "device-drivers/task/task-basic", script: "scripts/tasks-start.ts" },
  { id: "hardware/devices/toolbar", script: "scripts/devices-toolbar.ts" },
  {
    id: "device-drivers/ni/configure-device/ni-configure-device",
    script: "scripts/ni-configure-device.ts",
  },
  {
    id: "device-drivers/labjack/configure-device/lj-configure",
    script: "scripts/labjack-configure-device.ts",
  },
  {
    id: "device-drivers/ethercat/configure-device/ethercat-configure",
    script: "scripts/ethercat-configure-device.ts",
  },
  { id: "hardware/ni/analog-read-task", script: "scripts/ni-analog-read-task.ts" },
  { id: "hardware/ni/analog-write-task", script: "scripts/ni-analog-write-task.ts" },
  { id: "hardware/ni/counter-read-task", script: "scripts/ni-counter-read-task.ts" },
  { id: "hardware/ni/digital-read-task", script: "scripts/ni-digital-read-task.ts" },
  { id: "hardware/ni/digital-write-task", script: "scripts/ni-digital-write-task.ts" },
  { id: "hardware/labjack/read-task", script: "scripts/labjack-read-task.ts" },
  { id: "hardware/labjack/write-task", script: "scripts/labjack-write-task.ts" },
  { id: "hardware/ethercat/read-task", script: "scripts/ethercat-read-task.ts" },
  { id: "hardware/ethercat/write-task", script: "scripts/ethercat-write-task.ts" },
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
  // These run the real Driver against a server from fixtures/simulator.ts, on the port
  // the form shows.
  {
    id: "device-drivers/modbus/connect-server/modbus-connect",
    script: "scripts/modbus-connect-server.ts",
    serial: true,
  },
  {
    id: "device-drivers/opc-ua/connect-server/connect-server",
    script: "scripts/opcua-connect-server.ts",
    serial: true,
  },
  {
    id: "hardware/opc-ua/read-task",
    script: "scripts/opcua-read-task.ts",
    serial: true,
  },
  {
    id: "hardware/opc-ua/write-task",
    script: "scripts/opcua-write-task.ts",
    serial: true,
  },
  {
    id: "device-drivers/http/connect-server/connect-server",
    script: "scripts/http-connect-server.ts",
    serial: true,
  },
  {
    id: "hardware/http/read-task",
    script: "scripts/http-read-task.ts",
    serial: true,
  },
  {
    id: "hardware/http/write-task",
    script: "scripts/http-write-task.ts",
    serial: true,
  },
  {
    id: "hardware/modbus/read-task",
    script: "scripts/modbus-read-task.ts",
    serial: true,
  },
  {
    id: "hardware/modbus/write-task",
    script: "scripts/modbus-write-task.ts",
    serial: true,
  },
  {
    id: "device-drivers/pagerduty/alert-task/create",
    script: "scripts/pagerduty-alert-create.ts",
  },
  {
    id: "releases/0-57-0/panels",
    script: "scripts/release-0-57-0-panels.ts",
    themed: false,
  },
  { id: "releases/0-60-0/line-plots", script: "scripts/release-0-60-line-plots.ts" },
  {
    id: "releases/0-60-0/change-symbol",
    script: "scripts/release-0-60-change-symbol.ts",
  },
  {
    id: "releases/0-60-0/table-reorder",
    script: "scripts/release-0-60-table-reorder.ts",
  },
]);
