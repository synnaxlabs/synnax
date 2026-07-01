// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError } from "@synnaxlabs/client";

import { Export } from "@/feature/export";
import { LAYOUT_TYPE } from "@/platform/lineplot/layout";
import { Session } from "@/session";

export const extract: Export.Extractor = async (key, { store, client }) => {
  const name = Session.Layout.select(store.getState(), key)?.name;
  if (client == null) throw new DisconnectedError();
  const lp = await client.lineplots.retrieve({ key });
  return {
    data: JSON.stringify({ ...lp, type: LAYOUT_TYPE }),
    name: name ?? lp.name,
  };
};

export const useExport = () => Export.use(extract, "line plot");
