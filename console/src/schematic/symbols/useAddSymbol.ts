import { type Dispatch } from "@reduxjs/toolkit";
import { Flux, Schematic, Theming } from "@synnaxlabs/pluto";
import { id, type xy } from "@synnaxlabs/x";
import { useCallback } from "react";

import { addElement } from "@/schematic/slice";

export const useAddSymbol = (dispatch: Dispatch, layoutKey: string) => {
  const store = Flux.useStore<Schematic.Symbol.SubStore>();
  const theme = Theming.use();

  return useCallback(
    (key: string, position?: xy.XY, data?: any) => {
      let variant: Schematic.Symbol.Variant;
      let initialName: string | undefined;
      const symbol = store.schematicSymbols.get(data?.specKey as string);
      if (symbol != null) {
        variant = symbol.data.states.length === 1 ? "customStatic" : "customActuator";
        initialName = symbol.name;
      } else variant = key as Schematic.Symbol.Variant;
      const spec = Schematic.Symbol.REGISTRY[variant];
      const initialProps = spec.defaultProps(theme);
      if (symbol != null) {
        initialProps.specKey = key;
        initialProps.label.label = initialName;
        variant = "customStatic";
      }
      dispatch(
        addElement({
          key: layoutKey,
          elKey: id.create(),
          node: { zIndex: spec.zIndex, position },
          props: { key: variant, ...initialProps, ...data },
        }),
      );
    },
    [dispatch, layoutKey, theme],
  );
};
