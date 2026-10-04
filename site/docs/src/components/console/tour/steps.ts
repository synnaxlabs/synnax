// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** Where a group of steps sits in the Console. */
export interface Group {
  key: string;
  name: string;
}

/** What the Console shows while a step is active. */
export interface Scene {
  /** The left toolbar to open. */
  toolbar?: string;
  /** Whether the component toolbar is open. */
  component?: boolean;
  /** Whether the command palette is open. */
  palette?: boolean;
  /** The mosaic leaf to focus. */
  leaf?: string;
}

/** One stop of the tour. The page supplies its text. */
export interface Step {
  key: string;
  title: string;
  /** The fragment that links to the step, kept from the page's earlier headings. */
  anchor: string;
  group: string;
  /** The key that opens the part, when it has one. */
  trigger?: string;
  /** Selectors of the elements the step highlights, as one box around them all. */
  targets: string[];
  scene: Scene;
}

export const GROUPS: Group[] = [
  { key: "mosaic", name: "Mosaic" },
  { key: "left", name: "Left bar" },
  { key: "top", name: "Top bar" },
];

const slug = (title: string): string => title.toLowerCase().replaceAll(" ", "-");

const toolbar = (key: string, title: string, trigger: string): Step => ({
  key,
  title,
  anchor: slug(title),
  group: "left",
  trigger,
  targets: [`[data-toolbar="${key}"]`, `[data-drawer="${key}"]`],
  scene: { toolbar: key },
});

const leaf = (key: string, title: string): Step => ({
  key,
  title,
  anchor: slug(title),
  group: "mosaic",
  targets: [`[data-leaf="${key}"]`],
  scene: { leaf: key },
});

export const STEPS: Step[] = [
  {
    key: "mosaic",
    anchor: "parts-of-the-console",
    title: "Mosaic",
    group: "mosaic",
    targets: ['[data-part="mosaic"]'],
    scene: {},
  },
  leaf("schematic", "Schematic"),
  leaf("lineplot", "Line plot"),
  leaf("editor", "Arc editor"),
  {
    key: "project",
    anchor: "project-selector",
    title: "Project selector",
    group: "left",
    targets: ['[data-part="project"]'],
    scene: {},
  },
  {
    key: "search",
    anchor: "search-and-command-palette",
    title: "Search and command palette",
    group: "left",
    targets: ['[data-part="search"]', "[data-palette]"],
    scene: { palette: true },
  },
  {
    key: "toolbars",
    anchor: "toolbars",
    title: "Toolbars",
    group: "left",
    targets: ['[data-toolbar]:not([data-toolbar="component"])'],
    scene: {},
  },
  toolbar("channel", "Channels", "C"),
  toolbar("range", "Ranges", "R"),
  toolbar("workspace", "Projects", "W"),
  toolbar("device", "Devices", "D"),
  toolbar("task", "Tasks", "T"),
  toolbar("arc", "Arcs", "A"),
  toolbar("status", "Statuses", "S"),
  {
    key: "component",
    anchor: "component",
    title: "Component toolbar",
    group: "left",
    trigger: "V",
    targets: ['[data-toolbar="component"]', '[data-drawer="component"]'],
    scene: { component: true, leaf: "lineplot" },
  },
  {
    key: "panels",
    anchor: "panel-selector",
    title: "Panel selector",
    group: "top",
    targets: ['[data-part="panels"]'],
    scene: {},
  },
  {
    key: "help",
    anchor: "getting-help",
    title: "Help and updates",
    group: "top",
    targets: ['[data-part="help"]'],
    scene: {},
  },
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
