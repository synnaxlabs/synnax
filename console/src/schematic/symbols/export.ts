// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, type schematic } from "@synnaxlabs/client";

import { Export } from "@/export";

export interface ExportedSymbol {
  version: 1;
  type: "symbol";
  symbol: schematic.symbol.Symbol;
}

export const extract: Export.Extractor = async (key, { client }) => {
  if (client == null) throw new DisconnectedError();
  const symbol = await client.workspaces.schematic.symbols.retrieve({ key });
  const exportData: ExportedSymbol = {
    version: 1,
    type: "symbol",
    symbol,
  };
  return { data: JSON.stringify(exportData, null, 2), name: symbol.name };
};

export const useExport = () => Export.use(extract, "symbol");