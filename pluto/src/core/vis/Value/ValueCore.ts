// Copyrght 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useLayoutEffect } from "react";

import { Deep } from "@synnaxlabs/x";
import { z } from "zod";

import { Aether } from "@/core/aether/main";
import { useMemoCompare } from "@/core/memo";
import { Typography } from "@/core/std";
import { Theming } from "@/core/theming";
import { AetherValue } from "@/core/vis/Value/aether";

export const valueCoreProps = AetherValue.z
  .omit({ font: true })
  .partial({ color: true })
  .extend({ level: Typography.levelZ });

export type ValueCoreProps = z.input<typeof valueCoreProps>;

export const ValueCore = Aether.wrap<ValueCoreProps>(
  "ValueCore",
  ({ aetherKey, ...props }): ReactElement | null => {
    const theme = Theming.use();
    const font = Theming.useTypography(props.level);
    const memoProps = useMemoCompare(
      () => {
        return {
          font: font.toString(),
          ...props,
          color: theme.colors.text,
        };
      },
      ([prevProps], [nextProps]) => Deep.equal(prevProps, nextProps),
      [props, theme]
    );
    const [, , setState] = Aether.use({
      aetherKey,
      type: AetherValue.TYPE,
      schema: AetherValue.z,
      initialState: memoProps,
    });
    useLayoutEffect(() => setState((prev) => ({ ...prev, ...memoProps })), [memoProps]);
    return null;
  }
);
