// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift } from "@synnaxlabs/drift";
import { Haul, Mosaic, Tabs, Theming } from "@synnaxlabs/pluto";
import { theming } from "@synnaxlabs/pluto/ether";
import { location } from "@synnaxlabs/x";
import { z } from "zod";

export const placementLocationZ = z.union([
  z.literal("window"),
  z.literal("mosaic"),
  z.literal("modal"),
]);

/** The location options for placing a layout */
export type PlacementLocation = z.infer<typeof placementLocationZ>;

export const stateZ = z.object({
  windowKey: z.string(),
  key: z.string(),
  type: z.string(),
  name: z.string(),
  icon: z.string().optional(),
  location: placementLocationZ,
  windowProps: z.record(z.unknown()).optional(),
  tab: z.record(z.unknown()).optional(),
  args: z.unknown(),
});

/**
 * An extension of the drift window properties to allow for some custom layout tuning.
 */
export type WindowProps = Omit<Drift.WindowProps, "key" | "url"> & {
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

/**
 * Layout represents the properties of a layout currently rendered in the mosaic or in
 * an external window. The key of a layout must be unique.
 */
export interface State<A = any | undefined> {
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
   * layout will be placed in an external window. If the location is 'modal', the layout
   * will be placed in a modal window.
   */
  location: PlacementLocation;
  /**
   * Properties passed to the window constructor (if the location is 'window'). If the
   * location is 'mosaic', this property is ignored.
   */
  window?: WindowProps;
  /**
   * Properties used when the layout is placed in a tab. If the location is 'window' or
   * 'modal', these properties are ignored.
   */
  tab?: Partial<LayoutTabProps>;
  /**
   * A typically optional set of arguments to pass to the layout
   */
  args?: A;
}

export type RenderableLayout = Omit<State, "window">;

export const navDrawerEntryStateZ = z.object({
  activeItem: z.string().nullable(),
  menuItems: z.string().array(),
  size: z.number().optional(),
});

export type NavDrawerEntryState = z.infer<typeof navDrawerEntryStateZ>;

export const navDrawerLocationZ = z.union([
  z.literal("left"),
  z.literal("right"),
  z.literal("bottom"),
]);

export type NavDrawerLocation = z.infer<typeof navDrawerLocationZ>;

export const navDrawerStateZ = z.object({
  left: navDrawerEntryStateZ,
  right: navDrawerEntryStateZ,
  bottom: navDrawerEntryStateZ,
});

export type NavDrawerState = z.infer<typeof navDrawerStateZ>;

export const mainNavState = z.object({
  drawers: navDrawerStateZ,
});

export type MainNavState = z.infer<typeof mainNavState>;

export const partialNavState = z.object({
  drawers: navDrawerStateZ.partial(),
});

export type PartialNavState = z.infer<typeof partialNavState>;

export const mosaicStateZ = z.object({
  activeTab: z.string().nullable(),
  root: Mosaic.nodeZ,
  focused: z.string().optional().nullable().default(null),
});

export type MosaicState = z.infer<typeof mosaicStateZ>;

export const MAIN_LAYOUT: State = {
  name: "Main",
  key: "main",
  type: "main",
  location: "window",
  windowKey: Drift.MAIN_WINDOW,
  window: {
    navTop: false,
  },
};

export const ZERO_MOSAIC_STATE: MosaicState = {
  activeTab: null,
  focused: null,
  root: {
    key: 1,
    tabs: [],
  },
};

export const navStateZ = z.record(z.string(), partialNavState).and(
  z.object({
    main: mainNavState,
  }),
);

export const sliceStateZ = z.object({
  version: z.union([z.literal("0.0.0"), z.literal("0.1.0"), z.literal("0.2.0")]),
  activeTheme: z.string(),
  themes: z.record(z.string(), theming.specZ),
  layouts: z.record(z.string(), stateZ),
  hauling: Haul.draggingStateZ,
  mosaics: z.record(z.string(), mosaicStateZ),
  nav: navStateZ,
  alreadyCheckedGetStarted: z.boolean(),
});

export type SliceState = z.infer<typeof sliceStateZ>;

export const ZERO_SLICE_STATE: SliceState = {
  version: "0.0.0",
  activeTheme: "synnaxDark",
  themes: {
    synnaxDark: Theming.SYNNAX_THEMES.synnaxDark,
    synnaxLight: Theming.SYNNAX_THEMES.synnaxLight,
  },
  alreadyCheckedGetStarted: false,
  layouts: {
    main: MAIN_LAYOUT,
  },
  mosaics: {
    main: ZERO_MOSAIC_STATE,
  },
  hauling: Haul.ZERO_DRAGGING_STATE,
  nav: {
    main: {
      drawers: {
        left: {
          activeItem: null,
          menuItems: ["resources"],
        },
        right: {
          activeItem: null,
          menuItems: ["range", "task"],
        },
        bottom: {
          activeItem: null,
          menuItems: ["visualization"],
        },
      },
    },
  },
};
