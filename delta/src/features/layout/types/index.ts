import { ComponentType } from "react";

import type { WindowProps } from "@synnaxlabs/drift";

/** The location options for placing a layout */
export type LayoutPlacementLocation = "window" | "mosaic";

/**
 * Layout represents the properties of a layout currently rendered in the mosaic or in
 * an external window. The key of a layout must be unique.
 */
export interface Layout {
  /** A unique key for the layout */
  key: string;
  /**
   * The type of the layout. This is used to identify the correct renderer to
   * use for the layout.
   */
  type: string;
  /**
   * The title of the layout. This is either the window or tab title. NOTE: If the layout
   * is placed within a mosaic, this property is duplicated in the 'Tab' array of the
   * mosaic node. Ideally we'll change this in the future, but it's not worth it at
   * this point.
   */
  title: string;
  /**
   * Location defines the placement location of the layout. If the location is 'mosaic',
   * the layout will be placed in the central mosaic. If the location is 'window', the
   * layout will be placed in an external window.
   */
  location: LayoutPlacementLocation;
  /**
   * Properties passed to the window constructor (if the location is 'window'). If the
   * location is 'mosaic', this property is ignored.
   */
  window?: LayoutWindowProps;
}

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
export interface LayoutRendererProps {
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

/**
 * A React component that renders a layout for a given type. All layouts in state are
 * rendered by a layout renderer of a specific type.
 */
export type LayoutRenderer = ComponentType<LayoutRendererProps>;

/**
 * An extension of the drift window properties to allow for some custom layout tuning.
 */
export type LayoutWindowProps = Omit<WindowProps, "key" | "url"> & {
  /**
   * navTop is a flag that renders a simple, draggable navigation bar at the top of
   * the window. This is useful for layouts that may be rendered in a window or a
   * mosaic (as the mosaic shouldn't have a nav bar).
   */
  navTop?: boolean;
};
