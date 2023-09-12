// Copyrght 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ComponentPropsWithoutRef, type ReactElement } from "react";


import { Color } from "@/color";
import { CSS } from "@/css";

import "@/vis/tank/Tank.css";
import { dimensions } from "@synnaxlabs/x";

export interface TankProps extends Omit<ComponentPropsWithoutRef<"div">, "color"> {
  dimensions: dimensions.Dimensions;
  color: Color.Crude;
}

export const Tank = ({
  className,
  dimensions,
  style = {},
  color,
  ...props
}: TankProps): ReactElement => {
  if (color != null) style.borderColor = new Color.Color(color).rgbaCSS;
  return (
    <div
      className={CSS(className, CSS.B("tank"))}
      style={{ ...dimensions, ...style }}
      {...props}
    />
  );
};
