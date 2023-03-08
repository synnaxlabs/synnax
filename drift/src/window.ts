// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { XY, Dimensions } from "@synnaxlabs/x";

/** Represents the state of a window in it's lifecycle  */
export type WindowStage = "creating" | "created" | "closing" | "closed";

export const MAIN_WINDOW = "main";

/** State of a window managed by drift  */
export interface WindowState extends LabeledWindowProps {
  key: string;
  /** Lifecycle stage */
  stage: WindowStage;
  /** Number of active processes */
  processCount: number;
  reserved: boolean;
  error?: string;
  focusCount: number;
  centerCount: number;
}

/**
 * The properties to provide when creating a window.
 */
export interface WindowProps {
  /* A unique key for the window. If not provided, a unique key will be generated. */
  key: string;
  /** A custom runtime label for the window. */
  label?: string;
  /* The url to load in the window. */
  url?: string;
  /* The title of the window. */
  title?: string;
  /* Whether the window should be centered on the screen. */
  center?: boolean;
  /* The x and y coordinates of the window. */
  position?: XY;
  /* The dimensions of the window. */
  size?: Dimensions;
  /* The minimum dimensions of the window. */
  minSize?: Dimensions;
  /* The maximum dimensions of the window. */
  maxSize?: Dimensions;
  /* Whether the window should be resizable. */
  resizable?: boolean;
  /* Whether the window is fullscreen. */
  fullscreen?: boolean;
  /* Whether the window is focused. */
  focus?: boolean;
  /* Whether the window is maximized. */
  maximized?: boolean;
  /* Whether the window is visible. */
  visible?: boolean;
  minimized?: boolean;
  /* Decorations. Runtime specific. */
  decorations?: boolean;
  /* Whether to add the window to the task bar or not. Runtime specific. */
  skipTaskbar?: boolean;
  /* Whether to enable file drop. Runtime specific. */
  fileDropEnabled?: boolean;
  /* Whether the window is transparent. Runtime specific. */
  transparent?: boolean;
  alwaysOnTop?: boolean;
}

/* WindowProps but with a key */
export type LabeledWindowProps = Omit<WindowProps, "label" | "key"> & {
  label: string;
  key?: string;
};
