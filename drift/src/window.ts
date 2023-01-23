// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** Represents the state of a window in it's lifecycle  */
export type WindowState = 'creating' | 'created' | 'closing' | 'closed';

export const MAIN_WINDOW = 'main';

/** Properties of a window managed by drift  */
export interface Window {
  /** Lifecycle state */
  state: WindowState;
  /** Number of active processes */
  processCount: number;
  /** The props the  was created with */
  props: KeyedWindowProps;
}

/**
 * The properties to provide when creating a window.
 */
export interface WindowProps {
  /* A unique key for the window. If not provided, a unique key will be generated. */
  key?: string;
  /* The url to load in the window. */
  url?: string;
  /* The title of the window. */
  title?: string;
  /* Whether the window should be centered on the screen. */
  center?: boolean;
  /* X position of the window. */
  x?: number;
  /* Y position of the window. */
  y?: number;
  /* Width of the window. */
  width?: number;
  /* Height of the window. */
  height?: number;
  /* The minimum width of the window. */
  minWidth?: number;
  /* The minimum height of the window. */
  minHeight?: number;
  /* The maximum width of the window. */
  maxWidth?: number;
  /* The maximum height of the window. */
  maxHeight?: number;
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
  /* Decorations. Runtime specific. */
  decorations?: boolean;
  /* Whether to add the window to the task bar or not. Runtime specific. */
  skipTaskbar?: boolean;
  /* Whether to enable file drop. Runtime specific. */
  fileDropEnabled?: boolean;
  /* Whether the window is transparent. Runtime specific. */
  transparent?: boolean;
}

/* WindowProps but with a key */
export type KeyedWindowProps = Omit<WindowProps, 'key'> & { key: string };
