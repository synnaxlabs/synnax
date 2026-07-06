// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch, type UnknownAction } from "@reduxjs/toolkit";
import { type ComponentType } from "react";

/**
 * The props passed to a LayoutRenderer. Note that these props are minimal and only focus
 * on providing information that either allows the renderer to perform more data selections
 * from other locations in state OR allows the renderer to perform actions that may have
 * polymorphic behavior depending the layout location (i.e. closing a layout might remove
 * it from the mosaic or close the window, depending on the location).
 *
 * The goal here is to separate the rendering logic for a particular layout from its location
 * allowing us to mix and move layouts around the UI with ease.
 */
export interface RendererProps {
  /** The unique key of the layout. */
  layoutKey: string;
  visible: boolean;
  focused: boolean;
  /**
   * onClose should be called when the layout is ready to be closed. This function is
   * polymorphic and may have different behavior depending on the location of the layout.
   * For example, if the layout is in a window, onClose will close the window. If the
   * layout is in the mosaic, onClose will remove the layout from the mosaic.
   */
  onClose: () => void;
}

export interface OnCloseProps {
  dispatch: Dispatch<UnknownAction>;
  layoutKey: string;
}

/** The result returned by a layout's {@link UseName}. */
export interface NameHookResult {
  retrieve: () => void;
  /**
   * Called when the user renames the layout from the UI (e.g., editing the tab
   * in the mosaic). When undefined, the renderer falls back to dispatching
   * {@link rename} against the layout slice.
   */
  onRename: (name: string) => void;
}

/**
 * A hook bound to a layout {@link Renderer} that owns the name read/write path
 * for the layout. The hook is responsible for invoking {@link NameHookProps.onChange}
 * whenever its source-of-truth name updates and for persisting user-initiated
 * renames via {@link NameHookResult.onRename}. Display name is always read from
 * the layout slice; the hook keeps the slice in sync via `onChange`.
 */
export type UseName = (
  layoutKey: string,
  onChange: (name: string) => void,
) => NameHookResult;

/**
 * A React component that renders a layout for a given type. All layouts in state are
 * rendered by a layout renderer of a specific type. Renderers may optionally bind a
 * {@link UseName} via the `useName` property to take over the name read/write path
 * for layouts of their type.
 */
export type Renderer = ComponentType<RendererProps> & { useName?: UseName };

export interface ContextMenuProps {
  layoutKey: string;
}

export type ContextMenuRenderer = ComponentType<ContextMenuProps>;
