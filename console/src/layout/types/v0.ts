// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift } from "@synnaxlabs/drift";
import { Haul, Mosaic, Tabs, Theming } from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { z } from "zod";

export const VERSION = "0.0.0";

const placementLocationZ = z.enum(["window", "mosaic", "modal"]);

/**
 * The location options for placing a layout:
 * - window: Opens the layout in a new, standalone window separate from the main
 *   application.
 * - mosaic: Places the layout within the main window's mosaic tiling system, allowing
 *   it to be arranged and resized alongside other layouts
 * - modal: Displays the layout as a modal dialog that overlays the current window,
 *   typically used for temporary or focused interactions.
 */
type PlacementLocation = z.infer<typeof placementLocationZ>;

const windowPropsZ = Drift.windowPropsZ
  .omit({ key: true, url: true })
  .extend({ navTop: z.boolean().optional(), showTitle: z.boolean().optional() });

/**
 * An extension of the Drift window properties to allow for custom layout tuning.
 */
export type WindowProps = Omit<Drift.WindowProps, "key" | "url"> & {
  /**
   * navTop is a flag that renders a simple, draggable navigation bar at the top of the
   * window. This is useful for layouts that may be rendered in a window or a mosaic (as
   * the mosaic shouldn't have a nav bar).
   */
  navTop?: boolean;
  /**
   * showTitle is a flag that sets whether the name of the window will be displayed
   * as a title in the navigation bar. Only applies if navTop is true.
   */
  showTitle?: boolean;
};

const layoutTabPropsZ = Tabs.tabZ.pick({ closable: true, editable: true }).extend({
  tab: Tabs.tabZ,
  location: location.location.optional(),
  mosaicKey: z.number().optional(),
});

/**
 * The props passed to a LayoutTab. This is a subset of the properties of the
 * Tab interface for the Tabs component. This does not apply to window layoputs.
 */
interface LayoutTabProps extends Pick<Tabs.Tab, "closable" | "editable"> {
  tab: Tabs.Tab;
  location?: location.Location;
  mosaicKey?: number;
}

export const stateZ = z.object({
  windowKey: z.string(),
  key: z.string(),
  type: z.string(),
  name: z.string(),
  icon: z.string().optional(),
  location: placementLocationZ,
  windowProps: windowPropsZ.optional(),
  tab: layoutTabPropsZ.partial().optional(),
  args: z.unknown().optional(),
  excludeFromWorkspace: z.boolean().optional(),
  beta: z.boolean().default(false).optional(),
  unsavedChanges: z.boolean().default(false).optional(),
});

/**
 * Layout represents the properties of a layout currently rendered in the mosaic or in
 * an external window. The key of a layout must be unique.
 */
export interface State<A = unknown> {
  windowKey: string;
  /** A unique key for the layout */
  key: string;
  /**
   * The type of the layout. This is used to identify the correct renderer to use for
   * the layout.
   */
  type: string;
  /**
   * The name of the layout. This is either the window or tab name. NOTE: If the layout
   * is placed within a mosaic, this property is duplicated in the 'Tab' array of the
   * mosaic node. Ideally we'll change this in the future, but it's not worth it at this
   * point.
   */
  name: string;
  /**
   * The name of the icon associated with the layout.
   */
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
   * An optional set of arguments to pass to the layout.
   */
  args?: A;
  /**
   * excludeFromWorkspace is a flag that indicates whether the layout should be excluded
   * from the workspace. This is typically used for modal layouts.
   */
  excludeFromWorkspace?: boolean;
  /**
   * beta is a flag that indicates whether the layout should be marked with a beta tag.
   */
  beta?: boolean;
  /**
   * unsavedChanges is a flag that indicates whether the layout has unsaved changes.
   */
  unsavedChanges?: boolean;
  /**
   * loading is a flag that indicates whether the layout is loading.
   */
  loading?: boolean;
}

const mosaicStateZ = z.object({
  activeTab: z.string().nullable(),
  root: Mosaic.nodeZ,
  focused: z.string().nullable().default(null),
});

type MosaicState = z.infer<typeof mosaicStateZ>;

export const ZERO_MOSAIC_STATE: MosaicState = {
  activeTab: null,
  focused: null,
  root: { key: 1, tabs: [] },
};

export const navDrawerEntryStateZ = z.object({
  activeItem: z.string().nullable(),
  menuItems: z.string().array(),
  size: z.number().optional(),
  hover: z.boolean().optional(),
});

export type NavDrawerEntryState = z.infer<typeof navDrawerEntryStateZ>;

export type NavDrawerLocation = "left" | "right" | "bottom";

const navDrawerStateZ = z.object({
  left: navDrawerEntryStateZ,
  right: navDrawerEntryStateZ,
  bottom: navDrawerEntryStateZ,
});

const mainNavStateZ = z.object({ drawers: navDrawerStateZ });

const MAIN_LAYOUT_KEY = "main";

const partialNavStateZ = z.object({ drawers: navDrawerStateZ.partial() });

const navStateZ = z
  .record(z.string(), partialNavStateZ)
  .and(z.object({ [MAIN_LAYOUT_KEY]: mainNavStateZ }));

export type NavState = z.infer<typeof navStateZ>;

export const MAIN_LAYOUT: State = {
  name: "Main",
  key: MAIN_LAYOUT_KEY,
  type: MAIN_LAYOUT_KEY,
  location: "window",
  windowKey: Drift.MAIN_WINDOW,
  window: { navTop: false },
};

export const sliceStateZ = z.object({
  version: z.literal(VERSION),
  activeTheme: z.string(),
  themes: z.record(z.string(), Theming.themeZ),
  layouts: z.record(z.string(), stateZ),
  hauling: Haul.draggingStateZ,
  mosaics: z.record(z.string(), mosaicStateZ),
  nav: navStateZ,
  alreadyCheckedGetStarted: z.boolean(),
});

export type SliceState = z.infer<typeof sliceStateZ>;

export const ZERO_SLICE_STATE: SliceState = sliceStateZ.parse({
  version: VERSION,
  activeTheme: Theming.SYNNAX_DARK.key,
  themes: {
    [Theming.SYNNAX_DARK.key]: Theming.SYNNAX_THEMES.synnaxDark,
    [Theming.SYNNAX_LIGHT.key]: Theming.SYNNAX_THEMES.synnaxLight,
  },
  alreadyCheckedGetStarted: false,
  layouts: { main: MAIN_LAYOUT },
  mosaics: { main: ZERO_MOSAIC_STATE },
  hauling: Haul.ZERO_DRAGGING_STATE,
  nav: {
    main: {
      drawers: {
        left: {
          activeItem: null,
          menuItems: [
            "ontology",
            "channel",
            "range",
            "workspace",
            "device",
            "task",
            "user",
          ],
        },
        right: { activeItem: null, menuItems: [] },
        bottom: { activeItem: null, menuItems: ["visualization"] },
      },
    },
  },
});
