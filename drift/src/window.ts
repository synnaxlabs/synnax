// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { dimensions, xy } from "@synnaxlabs/x";
import { z } from "zod";

/** Represents the state of a window in it's lifecycle  */
export const windowStageZ = z.enum([
  "creating",
  "created",
  "closing",
  "closed",
  "reloading",
]);
export type WindowStage = z.infer<typeof windowStageZ>;

export const MAIN_WINDOW = "main";
export const PRERENDER_WINDOW = "prerender";

/** The properties to provide when creating a window. */
export const windowPropsZ = z.object({
  /** A unique key for the window. If not provided, a unique key will be created. */
  key: z.string(),
  /** The url to load in the window. */
  url: z.string().optional(),
  /** The title of the window. */
  title: z.string().optional(),
  /** Whether the window should be centered on the screen. */
  center: z.boolean().optional(),
  /** The x and y coordinates of the window. */
  position: xy.xyZ.optional(),
  /** The dimensions of the window. */
  size: dimensions.dimensionsZ.optional(),
  /** The minimum dimensions of the window. */
  minSize: dimensions.dimensionsZ.optional(),
  /** The maximum dimensions of the window. */
  maxSize: dimensions.dimensionsZ.optional(),
  /** Whether the window should be resizable. */
  resizable: z.boolean().optional(),
  /** Whether the window is fullscreen. */
  fullscreen: z.boolean().optional(),
  /** Whether the window is focused. */
  focus: z.boolean().optional(),
  /** Whether the window is maximized. */
  maximized: z.boolean().optional(),
  /** Whether the window is visible. */
  visible: z.boolean().optional(),
  /** Whether the window is minimized. */
  minimized: z.boolean().optional(),
  /** Decorations. Runtime specific. */
  decorations: z.boolean().optional(),
  /** Whether to add the window to the task bar or not. Runtime specific. */
  skipTaskbar: z.boolean().optional(),
  /** Whether to enable file drop. Runtime specific. */
  fileDropEnabled: z.boolean().optional(),
  /** Whether the window is transparent. Runtime specific. */
  transparent: z.boolean().optional(),
  /** Whether the window is always on top. Runtime specific. */
  alwaysOnTop: z.boolean().optional(),
});
export interface WindowProps extends z.infer<typeof windowPropsZ> {}

/** What drift tracks about a window on top of the properties it was created with. */
export const windowStateExtensionPropsZ = z.object({
  /** Lifecycle stage */
  stage: windowStageZ,
  /** Number of active processes */
  processCount: z.number(),
  /**
   * Whether the window has been reserved for use. If this value is false,
   * the window is a pre-forked window that is not currently in use.
   */
  reserved: z.boolean(),
  /**
   * If something went wrong while making changes to the window, the error
   * will be stored here.
   */
  error: z.string().optional(),
  /** Incremented to focus the window */
  focusCount: z.number(),
  /** Incremented to center the window */
  centerCount: z.number(),
  /**
   * Creation ordinal, assigned once when the window is reserved and never
   * reused. The main window is 1; pre-render windows have none until claimed.
   */
  ordinal: z.number().optional(),
});
export interface WindowStateExtensionProps extends z.infer<
  typeof windowStateExtensionPropsZ
> {}

/** State of a window managed by drift  */
export const windowStateZ = windowPropsZ.extend(windowStateExtensionPropsZ.shape);
export interface WindowState extends z.infer<typeof windowStateZ> {}

export const INITIAL_WINDOW_STATE: WindowStateExtensionProps = {
  stage: "creating",
  processCount: 0,
  reserved: false,
  focusCount: 0,
  centerCount: 0,
};

export const INITIAL_PRERENDER_WINDOW_STATE: WindowState = {
  ...INITIAL_WINDOW_STATE,
  key: PRERENDER_WINDOW,
  visible: false,
};

/**
 * Clears how the window was presented and where it was in its lifecycle when its
 * process ended. Identity and geometry survive.
 */
export const resetTransientState = (window: WindowState): WindowState => ({
  ...window,
  stage: "creating",
  processCount: 0,
  focusCount: 0,
  centerCount: 0,
  focus: undefined,
  minimized: undefined,
  fullscreen: undefined,
  error: undefined,
});
