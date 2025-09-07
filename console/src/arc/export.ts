// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError } from "@synnaxlabs/client";

import { select } from "@/arc/selectors";
import { type State } from "@/arc/slice";
import { Export } from "@/export";
import { Layout } from "@/layout";

export const extract: Export.Extractor = async (key, { store, client }) => {
  const storeState = store.getState();
  let state = select(storeState, key);
  let name = Layout.select(storeState, key)?.name;
  if (state == null || name == null) {
    if (client == null) throw new DisconnectedError();
    const arc = await client.arcs.retrieve({ key });
    state ??= {
      ...(arc as unknown as State),
      key: arc.key,
    };
    name ??= arc.key;
  }
  return { data: JSON.stringify(state), name };
};

export const useExport = () => Export.use(extract, "arc");
