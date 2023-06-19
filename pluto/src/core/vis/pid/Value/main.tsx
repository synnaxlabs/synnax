// Copyrght 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, memo, useCallback, useState } from "react";

import { Box, XY } from "@synnaxlabs/x";

import { Aether } from "@/core/aether/main";
import { ColorT } from "@/core/color";
import { useResize } from "@/core/hooks";
import { Pack, PackProps, Typography } from "@/core/std";
import { Theming } from "@/core/theming";
import {
  Value as WorkerValue,
  ValueState as WorkerValueState,
} from "@/core/vis/pid/Value/worker";
import { ComponentSize } from "@/util/component";

export interface ValueProps
  extends Omit<WorkerValueState, "font" | "color" | "box">,
    Omit<PackProps, "color"> {
  color?: ColorT;
  size?: ComponentSize;
}

export const Value = memo(
  ({ label, color, size = "medium", ...props }: ValueProps): ReactElement => {
    const theme = Theming.use();
    const {
      state: [, setState],
    } = Aether.use<WorkerValueState>(WorkerValue.TYPE, {
      font: Theming.font(theme, "p"),
      color: color ?? theme.colors.text,
      box: Box.ZERO,
      label,
      ...props,
    });

    const handleResize = useCallback(
      (box: Box) => {
        setState((prev) => ({ ...prev, box }));
      },
      [setState]
    );

    const resizeRef = useResize(handleResize, {});

    const [labelState, setLabelState] = useState(label);

    return (
      <Pack {...props} size={size} direction="y">
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
            height: (theme.typography.p.lineHeight + 1) * theme.sizes.base,
            padding: "1rem",
          }}
        ></div>
      </Pack>
    );
  }
);
Value.displayName = "Value";
