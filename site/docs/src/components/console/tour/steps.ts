// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Triggers } from "@synnaxlabs/lyra/triggers";

/** Where a group of steps sits in the Console. */
export interface Group {
  key: "mosaic" | "left" | "top";
  name: string;
}

/** One stop of the tour. The page supplies its text. */
export interface Step {
  key: string;
  title: string;
  /** The fragment that links to the step, kept from the page's earlier headings. */
  anchor: string;
  group: Group["key"];
  /** The key that opens the part, when it has one. */
  trigger?: Triggers.Key;
  /** The still of the Console that shows the step. */
  still: string;
  /** The regions of the still that the step highlights, as one box around them all. */
  regions: string[];
  /** Whether the still shows a dialog, which a selection outside of closes. */
  modal?: boolean;
}

export const GROUPS: Group[] = [
  { key: "mosaic", name: "Mosaic" },
  { key: "left", name: "Left bar" },
  { key: "top", name: "Top bar" },
];

const slug = (title: string): string => title.toLowerCase().replaceAll(" ", "-");

const part = (
  key: string,
  title: string,
  group: Group["key"],
  anchor = slug(title),
): Step => ({
  key,
  title,
  anchor,
  group,
  still: "mosaic",
  regions: [key],
});

const toolbar = (
  key: string,
  title: string,
  trigger: Triggers.Key,
  anchor = slug(title),
): Step => ({
  key,
  title,
  anchor,
  group: "left",
  trigger,
  still: key,
  regions: [key, "drawer"],
});

export const STEPS: Step[] = [
  part("mosaic", "Mosaic", "mosaic", "parts-of-the-console"),
  part("schematic", "Schematic", "mosaic"),
  part("lineplot", "Line plot", "mosaic"),
  part("editor", "Arc editor", "mosaic"),
  part("project", "Project selector", "left"),
  {
    key: "search",
    anchor: "search-and-command-palette",
    title: "Search and command palette",
    group: "left",
    still: "search",
    regions: ["search", "palette"],
    modal: true,
  },
  part("toolbars", "Toolbars", "left"),
  toolbar("channel", "Channels", "C"),
  toolbar("range", "Ranges", "R"),
  toolbar("workspace", "Projects", "W"),
  toolbar("device", "Devices", "D"),
  toolbar("task", "Tasks", "T"),
  toolbar("arc", "Arcs", "A"),
  toolbar("status", "Statuses", "S"),
  toolbar("component", "Component toolbar", "V", "component"),
  part("panels", "Panel selector", "top"),
  part("help", "Help and updates", "top", "getting-help"),
];

/**
 * Returns the step with the given key.
 * @throws {Error} if no step has the key.
 */
export const find = (key: string): Step => {
  const step = STEPS.find((s) => s.key === key);
  if (step == null) throw new Error(`unknown tour step: ${key}`);
  return step;
};
