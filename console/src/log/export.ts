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
import { LAYOUT_TYPE } from "@/log/layout";

export const extract: Export.Extractor = async (key, { store, client }) => {
  const name = Layout.select(store.getState(), key)?.name;
  if (client == null) throw new DisconnectedError();
  const l = await client.logs.retrieve({ key });
  return {
    data: JSON.stringify({ ...l, type: LAYOUT_TYPE }),
    name: name ?? l.name,
  };
};

export const useExport = () => Export.use(extract, "log");
