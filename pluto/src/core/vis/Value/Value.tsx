// Copyrght 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ComponentPropsWithoutRef, ReactElement, memo, useState } from "react";

import { Box } from "@synnaxlabs/x";

import { useResize } from "@/core/hooks";
import { Theming } from "@/core/theming";
import { ValueCore, ValueCoreProps } from "@/core/vis/Value/ValueCore";

import "@/core/vis/Value/Value.css";

export interface ValueProps
  extends Omit<ValueCoreProps, "box">,
    Omit<ComponentPropsWithoutRef<"span">, "color"> {}

export const Value = memo(
  ({ style, color, level = "p", ...props }: ValueProps): ReactElement => {
    const [box, setBox] = useState(Box.ZERO);
    const ref = useResize(setBox);
    const font = Theming.useTypography(level ?? "p");
    return (
      <div
        ref={ref}
        style={{
          height: (font.lineHeight + 2) * font.baseSize,
          border: "1px solid black",
          ...style,
        }}
        {...props}
      >
        {!box.isZero && <ValueCore box={box} color={color} level={level} {...props} />}
      </div>
    );
  }
);
Value.displayName = "Value";
