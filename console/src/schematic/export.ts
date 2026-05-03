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

export const extract: Export.Extractor = async (key, { store, client }) => {
  const storeState = store.getState();
  const name = Layout.select(storeState, key)?.name;
  if (client == null) throw new DisconnectedError();
  const schematic = await client.schematics.retrieve({ key });
  return {
    data: JSON.stringify({ ...schematic, type: LAYOUT_TYPE }),
    name: name ?? schematic.name,
  };
};

export const useExport = () => Export.use(extract, "schematic");
