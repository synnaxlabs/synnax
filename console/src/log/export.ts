// Copyright 2025 Synnax Labs, Inc.
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
import { select } from "@/log/selectors";
import { type State } from "@/log/slice";

export const extract: Export.Extractor = async (key, { store, client }) => {
  const storeState = store.getState();
  let state = select(storeState, key);
  let name = Layout.select(storeState, key)?.name;
  if (state == null || name == null) {
    if (client == null) throw new DisconnectedError();
    const log = await client.workspaces.logs.retrieve({ key });
    state ??= { ...(log.data as State), key: log.key };
    name ??= log.name;
  }
  return { data: JSON.stringify(state), name };
};

export const useExport = () => Export.use(extract, "log");
