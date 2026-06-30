// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, current, type PayloadAction } from "@reduxjs/toolkit";
import { Drift, MAIN_WINDOW } from "@synnaxlabs/drift";
import { Color, Haul, Mosaic, Tabs } from "@synnaxlabs/pluto";
import { deep, type direction, id, location } from "@synnaxlabs/x";
import { z } from "zod";

import { type BaseState } from "@/session/layout/usePlacer";
import { type State } from "@/session/store";

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
  location: location.locationZ.optional(),
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
  excludeFromProject: z.boolean().optional(),
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
   * excludeFromProject is a flag that indicates whether the layout should be excluded
   * from the project. This is typically used for modal layouts.
   */
  excludeFromProject?: boolean;
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

const MAIN_LAYOUT_KEY = "main";

export const MAIN_LAYOUT: State = {
  name: "Main",
  key: MAIN_LAYOUT_KEY,
  type: MAIN_LAYOUT_KEY,
  location: "window",
  windowKey: Drift.MAIN_WINDOW,
  window: { navTop: false },
};

export const sliceStateZ = z.object({
  version: z.literal(0).default(0),
  layouts: z.record(z.string(), stateZ).default({ [MAIN_LAYOUT_KEY]: MAIN_LAYOUT }),
  hauling: Haul.draggingStateZ.default(Haul.ZERO_DRAGGING_STATE),
  mosaics: z
    .record(z.string(), mosaicStateZ)
    .default({ [MAIN_LAYOUT_KEY]: ZERO_MOSAIC_STATE }),
  altKeyToKey: z.record(z.string(), z.string()).default({}),
  keyToAltKey: z.record(z.string(), z.string()).default({}),
  colorContext: Color.contextStateZ.default(Color.ZERO_CONTEXT_STATE),
});
export interface SliceState extends z.infer<typeof sliceStateZ> {}

export const ZERO_SLICE_STATE: SliceState = sliceStateZ.parse({});

/**
 * The name of the layout slice in a larger store.
 * NOTE: This must be the name of the slice in the store, or else all selectors will fail.
 */
export const SLICE_NAME = "layout";

/**
 * Represents a partial view of a larger store that contains the layout slice. This is
 * typically used for hooks that accept the entire store state as a parameter but only
 * need access to the layout slice.
 */
export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export const PERSIST_EXCLUDE = ["hauling"].map(
  (key) => `${SLICE_NAME}.${key}`,
) as Array<deep.Key<State>>;

/** Signature for the placeLayout action. */
export type PlacePayload = State;

/** Signature for the removeLayout action. */
export interface RemovePayload {
  keys: string[];
}

export interface MoveMosaicTabPayload {
  tabKey: string;
  windowKey?: string;
  key?: number;
  loc: location.Location;
  index?: number;
}

interface ResizeMosaicTabPayload {
  windowKey: string;
  key: number;
  size: number;
}

interface SplitMosaicNodePayload {
  windowKey: string;
  direction: direction.Direction;
  tabKey: string;
}

interface SelectMosaicTabPayload {
  tabKey: string;
}

interface RenamePayload {
  key: string;
  name: string;
}

interface SetFocusPayload {
  key: string | null;
  windowKey: string;
}

interface SetAltKeyPayload {
  key: string;
  altKey: string;
}

interface SetUnsavedChangesPayload {
  key: string;
  unsavedChanges: boolean;
}

interface SetHaulingPayload extends Haul.DraggingState {}

export interface SetProjectPayload {
  slice: SliceState;
}

interface SetArgsPayload<T = unknown> {
  key: string;
  args: T;
}

export interface SetColorContextPayload {
  state: Color.ContextState;
}

const purgeEmptyMosaics = (state: SliceState) => {
  Object.entries(state.mosaics).forEach(([key, mosaic]) => {
    if (key === MAIN_WINDOW || !Mosaic.isEmpty(mosaic.root)) return;
    delete state.mosaics[key];
    delete state.layouts[key];
  });
};

const select = (state: SliceState, key: string): State | null => {
  const layout = state.layouts[key];
  if (layout == null) {
    const altKey = state.altKeyToKey[key];
    if (altKey == null) return null;
    const altLayout = state.layouts[altKey];
    return altLayout ?? null;
  }
  return layout;
};

const layoutsToPreserve = (layouts: Record<string, State>): Record<string, State> =>
  Object.fromEntries(
    Object.entries(layouts).filter(
      ([, layout]) =>
        layout.location === "window" && layout.type !== MOSAIC_WINDOW_TYPE,
    ),
  );

const tabFromLayout = (layout: State): Tabs.Spec => ({
  closable: true,
  editable: layout.tab?.editable,
  icon: layout.icon,
  name: layout.name,
  tabKey: layout.key,
});

// Inserts a tab for every location:"mosaic" layout whose tab is missing
// from its claimed mosaic, falling back to the main mosaic when the window
// is gone. Persisted state can carry this shape of inconsistency, and
// without reconciliation the next attempt to re-open the layout throws
// "Tab not found" in place().
const reconcileMosaicLayouts = (state: SliceState) => {
  Object.values(state.layouts).forEach((layout) => {
    if (layout.location !== "mosaic") return;
    let target = state.mosaics[layout.windowKey];
    if (target == null) {
      layout.windowKey = MAIN_WINDOW;
      target = state.mosaics[MAIN_WINDOW];
      if (target == null) return;
    }
    if (Mosaic.findTabNode(target.root, layout.key) != null) return;
    target.root = Mosaic.insertTab(target.root, tabFromLayout(layout));
  });
};

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    place: (state, { payload: layout }: PayloadAction<PlacePayload>) => {
      const { location, tab } = layout;
      let key = layout.key;

      const prev = select(state, key);
      if (prev != null) {
        key = prev.key;
        layout.key = prev.key;
      }

      if (layout.type === MOSAIC_WINDOW_TYPE) state.mosaics[key] = ZERO_MOSAIC_STATE;

      // Clean up the source mosaic when leaving the mosaic location or
      // moving across windows. The source is keyed by prev.windowKey, not
      // by the incoming layout.windowKey.
      if (
        prev != null &&
        prev.location === "mosaic" &&
        (location !== "mosaic" || prev.windowKey !== layout.windowKey)
      ) {
        const prevMosaic = state.mosaics[prev.windowKey];
        if (prevMosaic != null)
          [prevMosaic.root] = Mosaic.removeTab(prevMosaic.root, key);
      }

      const mosaic = state.mosaics[layout.windowKey];
      const mosaicTab = tabFromLayout(layout);

      let mosaicKey = tab?.mosaicKey;
      // If we didn't explicitly specify a mosaic node to put the new tab in, and
      // the user has selected an active tab, we'll put the new tab in the same node
      // that the user has selected.
      if (mosaic?.activeTab != null && mosaicKey == null)
        mosaicKey = Mosaic.findTabNode(mosaic.root, mosaic.activeTab)?.key;

      // Decide insert vs. select/update by mosaic membership: a layout can
      // claim location "mosaic" without a matching mosaic node when
      // persisted state is inconsistent.
      if (location === "mosaic" && mosaic != null) {
        if (Mosaic.findTabNode(mosaic.root, key) != null)
          mosaic.root = Mosaic.updateTab(
            Mosaic.selectTab(mosaic.root, key),
            key,
            () => mosaicTab,
          );
        else
          mosaic.root = Mosaic.insertTab(
            mosaic.root,
            mosaicTab,
            tab?.location,
            mosaicKey,
          );
        mosaic.activeTab = key;
      }

      state.layouts[key] = layout;
      if (layout.type !== MOSAIC_WINDOW_TYPE) purgeEmptyMosaics(state);
    },
    setHauled: (state, { payload }: PayloadAction<SetHaulingPayload>) => {
      state.hauling = payload;
    },
    remove: (state, { payload: { keys } }: PayloadAction<RemovePayload>) => {
      keys.forEach((contentKey) => {
        const layout = select(state, contentKey);
        if (layout == null || layout.key == MAIN_WINDOW) return;
        const mosaic = state.mosaics[layout.windowKey];
        if (layout == null || mosaic == null) return;
        const { location } = layout;
        if (location === "mosaic")
          [mosaic.root, mosaic.activeTab] = Mosaic.removeTab(mosaic.root, layout.key);
        delete state.layouts[layout.key];
        state.mosaics[layout.windowKey] = mosaic;
        purgeEmptyMosaics(state);
      });
    },
    moveMosaicTab: (
      state,
      {
        payload: { tabKey, windowKey, key, loc, index },
      }: PayloadAction<MoveMosaicTabPayload>,
    ) => {
      const layout = select(state, tabKey);
      if (layout == null) return;
      const prevWindowKey = layout.windowKey;
      if (windowKey == null || prevWindowKey === windowKey) {
        // This is a redundant operation, so we leave everything as is.
        if (key == null) return;
        const mosaic = state.mosaics[prevWindowKey];
        [mosaic.root] = Mosaic.moveTab(mosaic.root, layout.key, loc, key, index);
        state.mosaics[prevWindowKey] = mosaic;
        return;
      }
      const prevMosaic = state.mosaics[prevWindowKey];
      [prevMosaic.root] = Mosaic.removeTab(prevMosaic.root, tabKey);
      state.mosaics[prevWindowKey] = prevMosaic;
      const mosaic = state.mosaics[windowKey];
      mosaic.activeTab ??= tabKey;
      state.layouts[layout.key].windowKey = windowKey;

      const mosaicTab = {
        closable: true,
        icon: layout.icon,
        ...layout.tab,
        name: layout.name,
        tabKey: layout.key,
      };

      mosaic.root = Mosaic.insertTab(mosaic.root, mosaicTab, loc, key);
      state.mosaics[windowKey] = mosaic;
      purgeEmptyMosaics(state);
    },
    setAltKey: (
      state,
      { payload: { key, altKey } }: PayloadAction<SetAltKeyPayload>,
    ) => {
      state.keyToAltKey[key] = altKey;
      state.altKeyToKey[altKey] = key;
    },
    splitMosaicNode: (
      state,
      {
        payload: { windowKey, direction, tabKey },
      }: PayloadAction<SplitMosaicNodePayload>,
    ) => {
      const mosaic = state.mosaics[windowKey];
      mosaic.root = Mosaic.split(mosaic.root, tabKey, direction);
      state.mosaics[windowKey] = mosaic;
    },
    selectMosaicTab: (
      state,
      { payload: { tabKey } }: PayloadAction<SelectMosaicTabPayload>,
    ) => {
      const layout = select(state, tabKey);
      if (layout == null) return;
      const { windowKey } = layout;
      const mosaic = state.mosaics[windowKey];
      if (mosaic.activeTab === tabKey) return;
      mosaic.root = Mosaic.selectTab(mosaic.root, layout.key);
      mosaic.activeTab = layout.key;
      state.mosaics[windowKey] = mosaic;
    },
    resizeMosaicTab: (
      state,
      { payload: { key, size, windowKey } }: PayloadAction<ResizeMosaicTabPayload>,
    ) => {
      const mosaic = state.mosaics[windowKey];
      mosaic.root = Mosaic.resizeNode(mosaic.root, key, size);
      state.mosaics[windowKey] = mosaic;
    },
    rename: (
      state,
      { payload: { key: tabKey, name } }: PayloadAction<RenamePayload>,
    ) => {
      const layout = select(state, tabKey);
      if (layout == null) return;
      const mosaic = state.mosaics[layout.windowKey];
      layout.name = name;
      mosaic.root = Mosaic.renameTab(mosaic.root, layout.key, name);
      state.mosaics[layout.windowKey] = mosaic;
    },
    setProject: (state, { payload: { slice } }: PayloadAction<SetProjectPayload>) => {
      // Mosaic.insertTab mutates tabs arrays in place; clone before
      // reconciling so the helper does not fight frozen nested objects
      // carried over from the previous store snapshot. Snapshot the draft
      // with current() first, since structuredClone cannot clone Immer's
      // draft Proxies.
      const s = current(state);
      const next = deep.copy({
        ...slice,
        layouts: {
          ...layoutsToPreserve(s.layouts),
          ...slice.layouts,
          main: MAIN_LAYOUT,
        },
        hauling: s.hauling,
      });
      reconcileMosaicLayouts(next);
      return next;
    },
    clearProject: (state) => ({
      ...ZERO_SLICE_STATE,
      layouts: {
        ...layoutsToPreserve(state.layouts),
        main: MAIN_LAYOUT,
      },
      hauling: state.hauling,
    }),
    setArgs: (state, { payload: { key, args } }: PayloadAction<SetArgsPayload>) => {
      const layout = select(state, key);
      if (layout == null) return;
      layout.args = args;
    },
    setFocus: (
      state,
      { payload: { key, windowKey } }: PayloadAction<SetFocusPayload>,
    ) => {
      if (key == null) {
        const mosaic = state.mosaics[windowKey];
        mosaic.focused = null;
        return;
      }
      const layout = select(state, key);
      if (layout == null) return;
      const mosaic = state.mosaics[layout.windowKey];
      mosaic.focused = key;
    },
    setColorContext: (state, { payload }: PayloadAction<SetColorContextPayload>) => {
      state.colorContext = payload.state;
    },
    setUnsavedChanges: (
      state,
      { payload }: PayloadAction<SetUnsavedChangesPayload>,
    ) => {
      const layout = select(state, payload.key);
      if (layout == null) return;
      layout.unsavedChanges = payload.unsavedChanges;

      if (layout.location === "mosaic") {
        const mosaic = state.mosaics[layout.windowKey];
        mosaic.root = Mosaic.updateTab(mosaic.root, layout.key, () => ({
          ...tabFromLayout(layout),
          unsavedChanges: payload.unsavedChanges,
        }));
      }
    },
  },
});

export const {
  place,
  setFocus,
  remove,
  moveMosaicTab,
  selectMosaicTab,
  resizeMosaicTab,
  setAltKey,
  splitMosaicNode,
  rename,
  setHauled,
  setProject,
  setColorContext,
  clearProject,
  setUnsavedChanges,
} = actions;

export const setArgs = <T>(pld: SetArgsPayload<T>): PayloadAction<SetArgsPayload<T>> =>
  actions.setArgs(pld) as PayloadAction<SetArgsPayload<T>>;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
export type Payload = Action["payload"];

export const MOSAIC_WINDOW_TYPE = "mosaicWindow";

export const createMosaicWindow = (window?: WindowProps): BaseState => ({
  key: `${MOSAIC_WINDOW_TYPE}-${id.create()}`,
  name: "Mosaic",
  type: MOSAIC_WINDOW_TYPE,
  location: "window",
  window: {
    ...window,
    size: { width: 800, height: 600 },
    navTop: false,
    visible: true,
    showTitle: false,
  },
});
