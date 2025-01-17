// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

import { Export } from "@/export";
import { Layout } from "@/layout";
import { select } from "@/lineplot/selectors";
import { type State } from "@/lineplot/slice";

export const extract: Export.Extractor = async (key, { store, client }) => {
  const storeState = store.getState();
  let state = select(storeState, key);
  let name = Layout.select(storeState, key)?.name;
  if (state == null || name == null) {
    if (client == null) throw new Error("Cannot reach cluster");
    const linePlot = await client.workspaces.linePlot.retrieve(key);
    state ??= { ...(linePlot.data as State), key: linePlot.key };
    name ??= linePlot.name;
  }
  return { data: JSON.stringify(state), name };
};

export const useExport = () => Export.use(extract, "lineplot");
