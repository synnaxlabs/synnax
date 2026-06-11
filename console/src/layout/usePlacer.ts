// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PayloadAction } from "@reduxjs/toolkit";
import { ontology, panel } from "@synnaxlabs/client";
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import { Flux, Panel, type Pluto, Status } from "@synnaxlabs/pluto";
import { id, uuid } from "@synnaxlabs/x";
import { type Dispatch, useCallback } from "react";
import { useDispatch, useStore } from "react-redux";

import { selectActivePanelKey, selectActiveTabKey } from "@/layout/selectors";
import { place, setActiveTab, type State } from "@/layout/slice";
import { type RootAction, type RootState, type RootStore } from "@/store";

export interface CreatorProps {
  dispatch: Dispatch<PayloadAction<unknown>>;
  store: RootStore;
  windowKey: string;
}

export interface BaseState<A = unknown>
  extends Omit<State<A>, "windowKey" | "key">, Partial<Pick<State<A>, "key">> {}

/** A function that creates a layout given a set of utilities. */
export interface Creator<A = unknown> {
  (props: CreatorProps): BaseState<A>;
}

export type PlacerArgs<A = unknown> = BaseState<A> | Creator<A>;

/** A function that places a layout using the given properties or creation func. */
export interface Placer<A = unknown> {
  (layout: PlacerArgs<A>): { windowKey: string; key: string };
}

// activeLeafPath resolves the path-derived key of the leaf to insert into: the leaf
// holding the active tab, falling back to the first leaf, then the root.
const activeLeafPath = (root: panel.Node, activeTab: string | null): number => {
  const fromActive = activeTab != null ? panel.tabLeafPath(root, activeTab) : null;
  return fromActive ?? panel.firstLeafPath(root) ?? panel.ROOT_PATH;
};

// tabFor builds the panel tab for a placed layout. A layout whose type is an ontology
// resource type references that resource (loaded from core by key); any other type is
// an inline view carrying its type, name, and opaque args (e.g. a task form keyed by
// its taskKey).
const tabFor = (layout: State): panel.Tab => {
  const resourceType = ontology.resourceTypeZ.safeParse(layout.type);
  if (resourceType.success)
    return {
      key: uuid.create(),
      resource: { type: resourceType.data, key: layout.key },
    };
  return {
    key: uuid.create(),
    view: {
      type: layout.type,
      name: layout.name,
      args: layout.args as panel.TabView["args"],
    },
  };
};

/**
 * useLayoutPlacer is a hook that returns a function that places a layout. It routes by
 * destination: document content (location "mosaic") is inserted into the active panel's
 * active leaf as a tab; window and modal layouts are session overlays stored in the
 * layout slice.
 *
 * @returns A layout placer that accepts either a layout object (key, type, title,
 * location, window props) or a layout creator function that receives utilities and
 * returns one. Prefer the object form; use the creator for dynamic creation.
 */
export const usePlacer = <A = unknown>(): Placer<A> => {
  const dispatch = useDispatch();
  const store = useStore<RootState, RootAction>();
  const windowKey = useSelectWindowKey();
  const fluxStore = Flux.useStore<Pluto.FluxStore>();
  const addStatus = Status.useAdder();
  const { dispatch: dispatchPanel } = Panel.useDispatch();
  return useCallback(
    (base) => {
      if (windowKey == null) throw new Error("windowKey is null");
      const layout =
        typeof base === "function" ? base({ dispatch, store, windowKey }) : base;
      const key = layout.key ?? id.create();
      const full = { ...layout, windowKey, key };
      // Document content goes into the active panel. The active panel is read from
      // the session slice and its tree from the Flux cache, so no dependency on the
      // panel module is needed.
      if (layout.location === "mosaic") {
        const state = store.getState();
        const panelKey = selectActivePanelKey(state);
        const cached = panelKey != null ? fluxStore.panels.get(panelKey) : null;
        if (panelKey == null || cached == null) {
          // No active panel (or its tree has not loaded yet) means document content
          // has nowhere to land. Surface the failure rather than storing a layout
          // the session slice can never render.
          addStatus({
            variant: "warning",
            message: `Cannot open ${full.name}: no active panel`,
          });
          return { windowKey, key };
        }
        const tab = tabFor(full);
        const targetLeaf = activeLeafPath(cached.root, selectActiveTabKey(state));
        dispatchPanel({
          key: panelKey,
          actions: [panel.insertTab({ tab, targetLeaf })],
        });
        dispatch(setActiveTab({ windowKey, key: tab.key }));
        return { windowKey, key };
      }
      dispatch(place(full));
      return { windowKey, key };
    },
    [dispatch, dispatchPanel, fluxStore, store, windowKey, addStatus],
  );
};
