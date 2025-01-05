// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

import { File } from "@/file";
import { Layout } from "@/layout";
import { select } from "@/table/selectors";
import { type State } from "@/table/slice";

export const extract: File.Extractor = async (key, { store, client }) => {
  const storeState = store.getState();
  let state = select(storeState, key);
  let name = Layout.select(storeState, key)?.name;
  if (state == null || name == null) {
    if (client == null) throw new Error("Cannot reach cluster");
    const table = await client.workspaces.table.retrieve(key);
    state ??= { ...(table.data as unknown as State), key: table.key };
    name ??= table.name;
  }
  const stateWithName = { ...state, name };
  return { file: JSON.stringify(stateWithName), name };
};

export const useExport = () => File.useExport(extract);
