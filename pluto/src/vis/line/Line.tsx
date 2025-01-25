// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Optional } from "@synnaxlabs/x";
import { type ReactElement, useLayoutEffect } from "react";

import { Aether } from "@/aether";
import { useMemoDeepEqualProps } from "@/memo";
import { line } from "@/vis/line/aether";

export interface LineProps
  extends Optional<Omit<line.State, "key">, "strokeWidth">,
    Aether.CProps {}

export const Line = ({ aetherKey, ...props }: LineProps): ReactElement | null => {
  const [, , setState] = Aether.use({
    aetherKey,
    type: line.Line.TYPE,
    schema: line.stateZ,
    initialState: props,
  });
  const memoProps = useMemoDeepEqualProps(props);
  useLayoutEffect(() => setState(memoProps), [memoProps]);
  return null;
};
