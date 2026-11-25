// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";

import { BUILT_INS } from "@/schematic/node/builtIns";
import { Custom } from "@/schematic/node/common/custom";
import { Symbol } from "@/schematic/symbol";

/** The symbol that a node draws, before the node's bindings apply. */
export interface Resolved {
  spec?: schematic.symbol.Spec;
  builtIn: boolean;
  missing: boolean;
  failed: boolean;
  invalid: boolean;
}

/** Resolves a symbol reference to its spec, reading a custom symbol from the Core. */
export const useResolved = (ref: schematic.symbol.Ref): Resolved => {
  const custom = Symbol.useResolved(ref.type === "custom" ? ref.key : null);
  switch (ref.type) {
    case "builtin": {
      const spec = BUILT_INS.get(ref.name);
      return {
        spec,
        builtIn: true,
        missing: spec == null,
        failed: false,
        invalid: false,
      };
    }
    case "custom": {
      const spec = custom.symbol?.data;
      // The check parses the markup, so it runs here, where it is cached on the spec.
      const invalid = spec != null && !Custom.isValidSVG(spec.drawing.markup);
      const { missing, failed } = custom;
      return { spec, builtIn: false, missing, failed, invalid };
    }
    case "missing":
      return { builtIn: false, missing: true, failed: false, invalid: false };
  }
};
