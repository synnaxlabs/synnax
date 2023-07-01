// Copyrght 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ComponentPropsWithoutRef, ReactElement } from "react";

import { Aether } from "@/core/aether/main";
import { Color, ColorT } from "@/core/color";
import { CSS } from "@/core/css";
import { Valve as WorkerValve, valveState } from "@/core/vis/Valve/aether";

import "@/core/vis/Valve/Valve.css";

export interface ValveProps extends Omit<ComponentPropsWithoutRef<"button">, "color"> {
  color?: ColorT;
  label?: string;
}

export const Valve = ({ color, style = {}, ...props }: ValveProps): ReactElement => {
  const [, { triggered, active }, setState] = Aether.use({
    type: WorkerValve.TYPE,
    schema: valveState,
    initialState: {
      triggered: false,
      active: false,
    },
  });

  const onClick = (): void =>
    setState((state) => ({ ...state, triggered: !state.triggered }));

  // @ts-expect-error
  if (color != null) style[CSS.var("base-color")] = new Color(color).rgbString;
  return (
    <button
      className={CSS(
        CSS.B("valve"),
        triggered && CSS.BM("valve", "triggered"),
        active && CSS.BM("valve", "active")
      )}
      onClick={onClick}
      style={style}
      {...props}
    >
      <svg width="100%" viewBox="0 0 102 50">
        <path d="M1 3.23317V46.7668C1 48.2529 2.56328 49.2199 3.89299 48.5564L50.107 25.4956C50.6693 25.215 51.3307 25.215 51.893 25.4956L98.107 48.5564C99.4367 49.2199 101 48.2529 101 46.7668V3.23317C101 1.74711 99.4367 0.780079 98.107 1.4436L51.893 24.5044C51.3307 24.785 50.6693 24.785 50.107 24.5044L3.893 1.4436C2.56329 0.78008 1 1.74711 1 3.23317Z" />
      </svg>
    </button>
  );
};
