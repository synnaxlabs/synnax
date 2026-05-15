// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch } from "@reduxjs/toolkit";
import { Theming } from "@synnaxlabs/lyra/theming";
import { Flux, Schematic } from "@synnaxlabs/pluto";
import { type xy } from "@synnaxlabs/x/xy";
import { useCallback } from "react";
import { useDispatch } from "react-redux";

import { addNode } from "@/schematic/slice";

export interface AddNodeProps {
  key: string;
  variant: Schematic.Node.Variant;
  specKey?: string;
  position?: xy.XY;
}

export const useAddNode = (layoutKey: string, dispatch?: Dispatch) => {
  const store = Flux.useStore<Schematic.Symbol.FluxSubStore>();
  const theme = Theming.use();
  const baseDispatch = useDispatch();
  const actualDispatch = dispatch ?? baseDispatch;

  return useCallback(
    ({ key, variant, position, specKey }: AddNodeProps) => {
      const spec = Schematic.Node.REGISTRY[variant];
      const config = spec.defaultConfig(theme);
      if (specKey != null)
        config.label = {
          ...config.label,
          label: store.schematicSymbols.get(key)?.name,
        };
      actualDispatch(
        addNode({
          key: layoutKey,
          node: { key, zIndex: spec.zIndex, position },
          config,
        }),
      );
    },
    [actualDispatch, layoutKey, theme],
  );
};
