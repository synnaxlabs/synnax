// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch, type UnknownAction } from "@reduxjs/toolkit";
import type { WindowProps as DriftWindowProps } from "@synnaxlabs/drift";
import { type Tabs } from "@synnaxlabs/pluto";
import { type location } from "@synnaxlabs/x";
import type { ComponentType } from "react";

/** The location options for placing a layout */
export type PlacementLocation = "window" | "mosaic" | "modal";

/**
 * Layout represents the properties of a layout currently rendered in the mosaic or in
 * an external window. The key of a layout must be unique.
 */
export interface State {
  windowKey: string;
  /** A unique key for the layout */
  key: string;
  /**
   * The type of the layout. This is used to identify the correct renderer to
   * use for the layout.
   */
  type: string;
  /**
   * The name of the layout. This is either the window or tab name. NOTE: If the layout
   * is placed within a mosaic, this property is duplicated in the 'Tab' array of the
   * mosaic node. Ideally we'll change this in the future, but it's not worth it at
   * this point.
   */
  name: string;
  /** */
  icon?: string;
  /**
   * Location defines the placement location of the layout. If the location is 'mosaic',
   * the layout will be placed in the central mosaic. If the location is 'window', the
   * layout will be placed in an external window.
   */
  location: PlacementLocation;
  /**
   * Properties passed to the window constructor (if the location is 'window'). If the
   * location is 'mosaic', this property is ignored.
   */
  window?: WindowProps;
  /**
   * Properties used when the layout is placed in a tab. If the location is 'window',
   * these properties are ignored.
   */
  tab?: Partial<LayoutTabProps>;
}

export type RenderableLayout = Omit<State, "window">;

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

/**
 * A React component that renders a layout for a given type. All layouts in state are
 * rendered by a layout renderer of a specific type.
 */
export type Renderer = ComponentType<RendererProps>;

/**
 * An extension of the drift window properties to allow for some custom layout tuning.
 */
export type WindowProps = Omit<DriftWindowProps, "key" | "url"> & {
  /**
   * navTop is a flag that renders a simple, draggable navigation bar at the top of
   * the window. This is useful for layouts that may be rendered in a window or a
   * mosaic (as the mosaic shouldn't have a nav bar).
   */
  navTop?: boolean;
  /**
   * showTitle is a flag that sets whether the name of the window will be displayed
   * as a title in the nav bar. Only applies if navTop is true.
   */
  showTitle?: boolean;
};

/**
 * The props passed to a LayoutTab. This is a subset of the properties of the
 * Tab interface for the Tabs component. This does not apply to window layoputs.
 */
export interface LayoutTabProps extends Pick<Tabs.Tab, "closable" | "editable"> {
  tab: Tabs.Tab;
  location?: location.Location;
  mosaicKey?: number;
}
