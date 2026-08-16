// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";

export const SYMBOL_FILE_FILTERS = [{ name: "JSON", extensions: ["json"] }];

export const isStatic = (symbol: schematic.symbol.Symbol): boolean =>
  symbol.data.variant === "static" || symbol.data.states.length === 1;
