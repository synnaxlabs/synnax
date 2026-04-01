// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { type Diagram, Flux, Schematic, Theming } from "@synnaxlabs/pluto";
import { id, xy, type record } from "@synnaxlabs/x";
import { useCallback } from "react";
import { z } from "zod";

const dropDataZ = z.object({
  specKey: schematic.symbol.keyZ,
});

export const useAddSymbol = (schematicKey: string) => {
  const store = Flux.useStore<Schematic.Symbol.FluxSubStore>();
  const theme = Theming.use();
  const { update: dispatch } = Schematic.useDispatch();

  return useCallback(
    (key: string, position?: xy.XY, data?: unknown) => {
      let variant: Schematic.Symbol.Variant;
      let initialName: string | undefined;
      let symbol: schematic.symbol.Symbol | undefined;
      const parsedData = dropDataZ.safeParse(data);
      if (parsedData.success)
        symbol = store.schematicSymbols.get(parsedData.data.specKey);
      if (symbol != null) {
        variant = symbol.data.states.length === 1 ? "customStatic" : "customActuator";
        initialName = symbol.name;
      } else variant = key as Schematic.Symbol.Variant;
      const spec = Schematic.Symbol.REGISTRY[variant];
      const initialProps = spec.defaultProps(theme);
      if (symbol != null) {
        initialProps.specKey = key;
        initialProps.label.label = initialName;
      }
      const nodeKey = id.create();
      const node: schematic.Node = {
        key: nodeKey,
        position: position ?? xy.ZERO,
        zIndex: spec.zIndex,
        measured: { width: 0, height: 0 },
      };
      const props: record.Unknown = {
        variant,
        ...initialProps,
        ...parsedData.data,
      };
      dispatch({
        key: schematicKey,
        actions: [schematic.addNode({ node, props })],
      });
    },
    [dispatch, schematicKey, theme, store],
  );
};
