// Copyrght 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type ComponentPropsWithoutRef,
  type ReactElement,
  memo,
  useState,
} from "react";

import { box } from "@synnaxlabs/x";

import { CSS } from "@/css";
import { useResize } from "@/hooks";
import { Theming } from "@/theming";
import { Core, type CoreProps } from "@/vis/value/Core";

export interface ValueProps
  extends Omit<CoreProps, "box">,
    Omit<ComponentPropsWithoutRef<"span">, "color"> {}

export const Value = memo(
  ({ style, color, level = "p", className, ...props }: ValueProps): ReactElement => {
    const [b, setBox] = useState(box.ZERO);
    const ref = useResize(setBox);
    const font = Theming.useTypography(level ?? "p");
    return (
      <div
        ref={ref}
        className={CSS(className, CSS.B("value"))}
        style={{
          height: (font.lineHeight + 2) * font.baseSize,
          border: "1px solid black",
          ...style,
        }}
        {...props}
      >
        {!box.isZero(b) && <Core box={b} color={color} level={level} {...props} />}
      </div>
    );
  },
);
Value.displayName = "Value";
