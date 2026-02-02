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
export type WindowStage = "creating" | "created" | "closing" | "closed" | "reloading";

export const MAIN_WINDOW = "main";
export const PRERENDER_WINDOW = "prerender";

export interface WindowStateExtensionProps {
  /** Lifecycle stage */
  stage: WindowStage;
  /** Number of active processes */
  processCount: number;
  /**
   * Whether the window has been reserved for use. If this value is false,
   * the window is a pre-forked window that is not currently in use.
   */
  reserved: boolean;
  /**
   * If something went wrong while making changes to the window, the error
   * will be stored here.
   */
  error?: string;
  /** Incremented to focus the window */
  focusCount: number;
  /** Incremented to center the window */
  centerCount: number;
}

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

/** State of a window managed by drift  */
export interface WindowState extends WindowProps, WindowStateExtensionProps {}

/**
 * The properties to provide when creating a window.
 */
export interface WindowProps {
  /* A unique key for the window. If not provided, a unique key will be created. */
  key: string;
  /* The url to load in the window. */
  url?: string;
  /* The title of the window. */
  title?: string;
  /* Whether the window should be centered on the screen. */
  center?: boolean;
  /* The x and y coordinates of the window. */
  position?: xy.XY;
  /* The dimensions of the window. */
  size?: dimensions.Dimensions;
  /* The minimum dimensions of the window. */
  minSize?: dimensions.Dimensions;
  /* The maximum dimensions of the window. */
  maxSize?: dimensions.Dimensions;
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
  /* Whether the window is minimized. */
  minimized?: boolean;
  /* Decorations. Runtime specific. */
  decorations?: boolean;
  /* Whether to add the window to the task bar or not. Runtime specific. */
  skipTaskbar?: boolean;
  /* Whether to enable file drop. Runtime specific. */
  fileDropEnabled?: boolean;
  /* Whether the window is transparent. Runtime specific. */
  transparent?: boolean;
  /* Whether the window is always on top. Runtime specific. */
  alwaysOnTop?: boolean;
}

export const windowPropsZ = z.object({
  key: z.string(),
  url: z.string().optional(),
  title: z.string().optional(),
  center: z.boolean().optional(),
  position: xy.xyZ.optional(),
  size: dimensions.dimensionsZ.optional(),
  minSize: dimensions.dimensionsZ.optional(),
  maxSize: dimensions.dimensionsZ.optional(),
  resizable: z.boolean().optional(),
  fullscreen: z.boolean().optional(),
  focus: z.boolean().optional(),
  maximized: z.boolean().optional(),
  visible: z.boolean().optional(),
  minimized: z.boolean().optional(),
  decorations: z.boolean().optional(),
  skipTaskbar: z.boolean().optional(),
  fileDropEnabled: z.boolean().optional(),
  transparent: z.boolean().optional(),
  alwaysOnTop: z.boolean().optional(),
});
