// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useEffect } from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { useMemoDeepEqualProps } from "@/memo";
import { light } from "@/vis/light/aether";

export interface UseProps extends Pick<z.input<typeof light.lightStateZ>, "source"> {
  aetherKey: string;
}

export interface UseReturn
  extends Pick<z.output<typeof light.lightStateZ>, "enabled"> {}

export const use = ({ aetherKey, source }: UseProps): UseReturn => {
  const memoProps = useMemoDeepEqualProps({ source });
  const [, { enabled }, setState] = Aether.use({
    aetherKey,
    type: light.Light.TYPE,
    schema: light.lightStateZ,
    initialState: {
      enabled: false,
      ...memoProps,
    },
  });
  useEffect(
    () => setState((state) => ({ ...state, ...memoProps })),
    [memoProps, setState],
  );
  return { enabled };
};
