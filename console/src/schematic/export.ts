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
import { LAYOUT_TYPE } from "@/schematic/Schematic";
import { selectOptional } from "@/schematic/selectors";
import { ZERO_STATE } from "@/schematic/slice";

export const extract: Export.Extractor = async (key, { store, client }) => {
  const storeState = store.getState();
  let state = selectOptional(storeState, key);
  let name = Layout.select(storeState, key)?.name;
  if (state == null || name == null) {
    if (client == null) throw new DisconnectedError();
    const schematic = await client.schematics.retrieve({ key });
    state ??= {
      ...ZERO_STATE,
      key: schematic.key,
      snapshot: schematic.snapshot,
      authority: schematic.authority,
      legend: schematic.legend,
      nodes: schematic.nodes,
      edges: schematic.edges,
      props: schematic.props,
    };
    name ??= schematic.name;
  }
  return { data: JSON.stringify({ ...state, type: LAYOUT_TYPE }), name };
};

export const useExport = () => Export.use(extract, "schematic");
