// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { workspace } from "@synnaxlabs/client";
import { Access, Flux, type Pluto, Synnax, Workspace } from "@synnaxlabs/pluto";
import { deep, TimeSpan } from "@synnaxlabs/x";
import { useCallback, useEffect, useRef } from "react";
import { useStore } from "react-redux";

import { Layout } from "@/layout";
import { type RootState } from "@/store";
import { purgeExcludedLayouts } from "@/workspace/purgeExcludedLayouts";
import { selectActiveKey } from "@/workspace/selectors";

export const useSyncLayout = (): void => {
  const store = useStore<RootState>();
  const fluxStore = Flux.useStore<Pluto.FluxStore>();
  const client = Synnax.use();
  const prevSyncRef = useRef<unknown>(null);
  const sync = Workspace.useSaveLayout({
    debounce: TimeSpan.milliseconds(250).milliseconds,
    beforeUpdate: useCallback(async () => {
      const s = store.getState();
      const key = selectActiveKey(s);
      if (key == null) return false;
      if (!Access.editGranted({ id: workspace.ontologyID(key), store: fluxStore, client })) return false;
      const layoutSlice = Layout.selectSliceState(s);
      if (deep.equal(prevSyncRef.current, layoutSlice)) return false;
      prevSyncRef.current = layoutSlice;
      const layout = purgeExcludedLayouts(layoutSlice);
      return { key, layout };
    }, [store, fluxStore, client]),
  });

  useEffect(() => store.subscribe(() => sync.update({ key: "", layout: {} })), []);
};
