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
  {
    id: "console/line-plots/properties-tab",
    script: "scripts/line-plot-properties.ts",
  },
  { id: "console/tables/create", script: "scripts/table-create.ts" },
  { id: "console/schematics/create", script: "scripts/schematic-create.ts" },
  { id: "console/channels/create", script: "scripts/channels-create.ts" },
  { id: "console/channels/alias", script: "scripts/channels-alias.ts" },
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
]);
