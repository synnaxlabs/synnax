// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { z } from "zod";

export interface ExportedSymbol extends schematic.symbol.Symbol {}

const manifestZ = z.object({
  file: z.string(),
  key: z.string(),
  name: z.string(),
});

export interface SymbolManifest extends z.infer<typeof manifestZ> {}

export const groupManifestZ = z.object({
  version: z.literal(1),
  type: z.literal("symbol_group"),
  name: z.string(),
  symbols: manifestZ.array(),
});

export interface GroupManifest extends z.infer<typeof groupManifestZ> {}

export const SYMBOL_FILE_FILTERS = [{ name: "JSON", extensions: ["json"] }];

export const isStatic = (symbol: schematic.symbol.Symbol): boolean =>
  symbol.data.variant === "static" || symbol.data.states.length === 1;
