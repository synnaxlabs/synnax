// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type resource } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";

/** Version control as an access control object. */
export const RESOURCE_ID: resource.ID = { type: "builtin", key: "version_control" };

/** The key of the version control toolbar. */
export const TOOLBAR_KEY = "version-control";

const TYPE_ICONS: Record<string, Icon.FC> = {
  arc: Icon.Arc,
  arc_directory: Icon.Group,
  channel: Icon.Channel,
  group: Icon.Group,
  lineplot: Icon.LinePlot,
  log: Icon.Log,
  panel: Icon.Panel,
  project: Icon.Project,
  schematic: Icon.Schematic,
  schematic_symbol: Icon.Schematic,
  table: Icon.Table,
  task: Icon.Task,
};

/** Returns the icon for a resource type, or a generic one for an unknown type. */
export const typeIcon = (type: string): Icon.FC => TYPE_ICONS[type] ?? Icon.Resources;

/** Returns the compact resource ID string the Core uses for version control. */
export const idString = ({ type, key }: resource.ID): string => `${type}:${key}`;

/** Returns the count with the word, plural unless the count is one: "3 changes". */
export const count = (n: number, word: string): string =>
  `${n} ${word}${n === 1 ? "" : "s"}`;
