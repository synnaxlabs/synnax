// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NULL_CLIENT_ERROR } from "@/errors";
import { Export } from "@/export";
import { Layout } from "@/layout";
import { select } from "@/slate/selectors";
import { type State } from "@/slate/slice";

export const extract: Export.Extractor = async (key, { store, client }) => {
  const storeState = store.getState();
  let state = select(storeState, key);
  let name = Layout.select(storeState, key)?.name;
  if (state == null || name == null) {
    if (client == null) throw NULL_CLIENT_ERROR;
    const slate = await client.workspaces.slate.retrieve(key);
    state ??= {
      ...(slate.data as unknown as State),
      snapshot: slate.snapshot,
      key: slate.key,
    };
    name ??= slate.name;
  }
  return { data: JSON.stringify(state), name };
};

export const useExport = () => Export.use(extract, "slate");
