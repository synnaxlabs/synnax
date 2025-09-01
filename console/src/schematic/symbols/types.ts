// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { z } from "zod";

export interface ExportedSymbol {
  version: 1;
  type: "symbol";
  symbol: schematic.symbol.Symbol;
}

export const exportedSymbolZ = z.object({
  version: z.literal(1),
  type: z.literal("symbol"),
  symbol: schematic.symbol.symbolZ,
});

export interface SymbolManifest {
  file: string;
  key: string;
  name: string;
}

export interface GroupManifest {
  version: 1;
  type: "symbol-group";
  name: string;
  symbols: SymbolManifest[];
}

export const groupManifestZ = z.object({
  version: z.literal(1),
  type: z.literal("symbol-group"),
  name: z.string(),
  symbols: z.array(
    z.object({
      file: z.string(),
      key: z.string(),
      name: z.string(),
    }),
  ),
});

export const SYMBOL_FILE_FILTERS = [{ name: "JSON", extensions: ["json"] }];