// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** What the copy of the Console shows while a step is active. */
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

/** How the copy of the Console shows one step. */
export interface Shot {
  /** Selectors of the elements the step highlights, as one box around them all. */
  targets: string[];
  scene: Scene;
}

const toolbar = (key: string): Shot => ({
  targets: [`[data-toolbar="${key}"]`, `[data-drawer="${key}"]`],
  scene: { toolbar: key },
});

const leaf = (key: string): Shot => ({
  targets: [`[data-leaf="${key}"]`],
  scene: { leaf: key },
});

/** The shot of each tour step, by step key. */
export const SHOTS: Record<string, Shot> = {
  mosaic: { targets: ['[data-part="mosaic"]'], scene: {} },
  schematic: leaf("schematic"),
  lineplot: leaf("lineplot"),
  editor: leaf("editor"),
  project: { targets: ['[data-part="project"]'], scene: {} },
  search: {
    targets: ['[data-part="search"]', "[data-palette]"],
    scene: { palette: true },
  },
  toolbars: { targets: ['[data-toolbar]:not([data-toolbar="component"])'], scene: {} },
  channel: toolbar("channel"),
  range: toolbar("range"),
  workspace: toolbar("workspace"),
  device: toolbar("device"),
  task: toolbar("task"),
  arc: toolbar("arc"),
  status: toolbar("status"),
  component: {
    targets: ['[data-toolbar="component"]', '[data-drawer="component"]'],
    scene: { component: true, leaf: "lineplot" },
  },
  panels: { targets: ['[data-part="panels"]'], scene: {} },
  help: { targets: ['[data-part="help"]'], scene: {} },
};
