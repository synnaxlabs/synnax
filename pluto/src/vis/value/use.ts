// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { text } from "@synnaxlabs/x";
import { useEffect } from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { useMemoDeepEqual } from "@/memo";
import { Value } from "@/vis/value/aether/value";

export const basePropsZ = Value.z
  .partial({ color: true })
  .extend({ level: text.levelZ.optional() });

export interface UseProps extends z.input<typeof basePropsZ> {
  aetherKey: string;
}

export const use = ({
  aetherKey,
  box,
  telem,
  color,
  level = "small",
  backgroundTelem,
  location,
  stalenessColor,
  stalenessTimeout,
  valueBackgroundOverScan,
  valueBackgroundShift,
  clip,
  borderRadius,
}: UseProps): void => {
  const memoProps = useMemoDeepEqual({
    box,
    telem,
    color,
    level,
    backgroundTelem,
    stalenessColor,
    stalenessTimeout,
    location,
    valueBackgroundOverScan,
    valueBackgroundShift,
    clip,
    borderRadius,
  });
  const [, , setState] = Aether.use({
    aetherKey,
    type: Value.TYPE,
    schema: Value.z,
    initialState: memoProps,
  });
  useEffect(() => setState((prev) => ({ ...prev, ...memoProps })), [memoProps]);
};
