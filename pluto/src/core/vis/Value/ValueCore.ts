// Copyrght 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, memo, useLayoutEffect, useMemo } from "react";

import { z } from "zod";

import { Aether } from "@/core/aether/main";
import { Typography } from "@/core/std";
import { Theming } from "@/core/theming";
import { AetherValue } from "@/core/vis/Value/aether";

export const valueCoreProps = AetherValue.stateZ
  .omit({ font: true })
  .partial({ color: true })
  .extend({ level: Typography.levelZ });

export type ValueCoreProps = z.input<typeof valueCoreProps>;

export const ValueCore = memo(
  ({ color, level = "p", ...props }: ValueCoreProps): ReactElement | null => {
    const theme = Theming.use();
    const font = Theming.useTypography(level);
    const memoProps = useMemo(
      () => ({
        font: font.toString(),
        color: color ?? theme.colors.text,
        ...props,
      }),
      [props, color, theme]
    );
    const [, , setState] = Aether.useStateful({
      type: AetherValue.TYPE,
      schema: AetherValue.stateZ,
      initialState: memoProps,
    });
    useLayoutEffect(() => setState((prev) => ({ ...prev, ...memoProps })), [memoProps]);
    return null;
  }
);
ValueCore.displayName = "ValueCore";
