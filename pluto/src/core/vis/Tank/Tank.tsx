// Copyrght 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ComponentPropsWithoutRef, ReactElement } from "react";

import { CrudeDimensions } from "@synnaxlabs/x";

import { Aether } from "@/core/aether/main";
import { Color, ColorT } from "@/core/color";
import { CSS } from "@/core/css";

import "@/core/vis/Tank/Tank.css";

export interface TankProps extends Omit<ComponentPropsWithoutRef<"div">, "color"> {
  dimensions: CrudeDimensions;
  color: ColorT;
}

export const Tank = Aether.wrap<TankProps>(
  "Tank",
  ({ aetherKey, className, dimensions, style = {}, color, ...props }): ReactElement => {
    if (color != null) style.borderColor = new Color(color).rgbaCSS;
    return (
      <div
        className={CSS(className, CSS.B("tank"))}
        style={{ ...dimensions, ...style }}
        {...props}
      />
    );
  }
);
