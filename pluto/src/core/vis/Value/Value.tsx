// Copyrght 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, memo, useCallback, useLayoutEffect, useState } from "react";

import { Box, XY } from "@synnaxlabs/x";
import { z } from "zod";

import { Aether } from "@/core/aether/main";
import { ColorT } from "@/core/color";
import { CSS } from "@/core/css";
import { useResize } from "@/core/hooks";
import { Pack, PackProps, Typography } from "@/core/std";
import { Theming } from "@/core/theming";
import { AetherValue } from "@/core/vis/Value/aether";
import { ComponentSize } from "@/util/component";

import "@/core/vis/Value/Value.css";

export interface ValueProps
  extends Omit<z.input<typeof AetherValue.stateZ>, "font" | "color" | "box">,
    Omit<PackProps, "color"> {
  color?: ColorT;
  size?: ComponentSize;
  label?: string;
  selected?: boolean;
}

export const Value = memo(
  ({
    label,
    color,
    position = XY.ZERO,
    size = "medium",
    selected = false,
    className,
    ...props
  }: ValueProps): ReactElement => {
    const theme = Theming.use();
    const [, , setState] = Aether.useStateful({
      type: AetherValue.TYPE,
      schema: AetherValue.stateZ,
      initialState: {
        font: Theming.font(theme, "p"),
        color: color ?? theme.colors.text,
        box: Box.ZERO,
        position,
        ...props,
      },
    });

    useLayoutEffect(() => {
      setState((prev) => ({ ...prev, position }));
    }, [position, setState]);

    const handleResize = useCallback(
      (box: Box) => {
        setState((prev) => ({ ...prev, box }));
      },
      [setState]
    );

    const resizeRef = useResize(handleResize, {
      triggers: position != null ? ["resizeX", "resizeY"] : undefined,
    });

    const [labelState, setLabelState] = useState(label);

    return (
      <Pack
        {...props}
        size={size}
        direction="y"
        className={CSS(className, selected && CSS.BM("value", "selected"))}
      >
        {label != null && (
          <Typography.Text.Editable
            level="p"
            color={color}
            style={{ padding: "1rem", textAlign: "center" }}
            value={labelState}
            onChange={setLabelState}
          />
        )}
        <div
          ref={resizeRef}
          style={{
            height: (theme.typography.p.lineHeight + 2) * theme.sizes.base,
          }}
        ></div>
      </Pack>
    );
  }
);
Value.displayName = "Value";
