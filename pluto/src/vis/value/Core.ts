// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useLayoutEffect } from "react";

import { type z } from "zod";

import { Aether } from "@/aether";
import { useMemoDeepEqualProps } from "@/memo";
import { Text } from "@/text";
import { Theming } from "@/theming";
import { Value } from "@/vis/value/aether/value";

export const corePropsZ = Value.z
  .omit({ font: true })
  .partial({ color: true })
  .extend({ level: Text.levelZ });

export type CoreProps = z.input<typeof corePropsZ>;

export const Core = Aether.wrap<CoreProps>(
  "ValueCore",
  ({
    aetherKey,
    box,
    telem,
    units,
    color,
    precision,
    width,
    level,
  }): ReactElement | null => {
    const font = Theming.useTypography(level);
    const memoProps = useMemoDeepEqualProps({
      box,
      telem,
      units,
      color,
      precision,
      width,
      font: font.toString(),
    });

    const [, , setState] = Aether.use({
      aetherKey,
      type: Value.TYPE,
      schema: Value.z,
      initialState: memoProps,
    });
    useLayoutEffect(() => setState((prev) => ({ ...prev, ...memoProps })), [memoProps]);
    return null;
  },
);
