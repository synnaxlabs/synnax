// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { Flux, type Pluto, Schematic, Status, Synnax } from "@synnaxlabs/pluto";
import { useCallback, useMemo } from "react";
import { useStore } from "react-redux";

import { create } from "@/component/schematic/layouts/layout";
import { Layout } from "@/layout";
import { Session } from "@/session";
import { type RootState } from "@/session/store";

type SchematicRetriever = (key: string) => Promise<schematic.Schematic>;

const navigateToLinkedSchematic = async (
  retrieve: SchematicRetriever,
  page: string,
  placeLayout: Layout.Placer,
): Promise<void> => {
  const s = await retrieve(page);
  placeLayout(create({ key: s.key, name: s.name }));
};

type NodeClickHandler = (nodeId: string, dblClick: boolean) => void;

export const useHandleNodeClickAction = (layoutKey: string): NodeClickHandler => {
  const store = useStore<RootState>();
  const client = Synnax.use();
  const fluxStore = Flux.useStore<Pluto.FluxStore>();
  const retrieve: SchematicRetriever | null = useMemo(
    () =>
      client != null
        ? (key: string) =>
            Schematic.retrieveSingle({ store: fluxStore, client, query: { key } })
        : null,
    [fluxStore, client],
  );
  const handleError = Status.useErrorHandler();
  const placeLayout = Layout.usePlacer();

  return useCallback(
    (nodeId: string, dblClick: boolean) => {
      const storeState = store.getState();
      const ui = Session.Schematic.selectState({ state: storeState, key: layoutKey });
      if (ui == null || ui.editable || retrieve == null) return;
      const config = fluxStore.schematics.get(layoutKey)?.configs?.[nodeId];
      if (
        config?.variant !== "offPageReference" ||
        typeof config.page !== "string" ||
        config.page.length === 0
      )
        return;
      const dblClickNav = config.dblClickNav !== false;
      if (dblClick !== dblClickNav) return;
      const { page } = config;
      const labelObj = config.label as { label?: string } | undefined;
      const label = labelObj?.label;
      const name = label != null && label.length > 0 ? label : "Referenced schematic";
      handleError(
        () => navigateToLinkedSchematic(retrieve, page, placeLayout),
        `Schematic "${name}" not found`,
      );
    },
    [store, layoutKey, retrieve, placeLayout, handleError, fluxStore],
  );
};
