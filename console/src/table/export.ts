// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError } from "@synnaxlabs/client";

import { Export } from "@/export";
import { Layout } from "@/layout";
import { select } from "@/table/selectors";
import { type State } from "@/table/slice";
import { LAYOUT_TYPE } from "@/table/Table";

export const extract: Export.Extractor = async (key, { store, client }) => {
  const storeState = store.getState();
  let state = select(storeState, key);
  let name = Layout.select(storeState, key)?.name;
  if (state == null || name == null) {
    if (client == null) throw new DisconnectedError();
    const table = await client.tables.retrieve({ key });
    state ??= { ...(table.data as State), key: table.key };
    name ??= table.name;
  }
  return { data: JSON.stringify({ ...state, type: LAYOUT_TYPE }), name };
};

export const useExport = () => Export.use(extract, "table");
