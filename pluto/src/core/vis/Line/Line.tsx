// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, memo, useEffect } from "react";

import { Optional } from "@synnaxlabs/x";

import { Aether } from "@/core/aether/main";
import { lineState, LineState } from "@/core/vis/Line/core";
import { LineGL } from "@/core/vis/Line/LineGL";

export interface LineProps extends Optional<Omit<LineState, "key">, "strokeWidth"> {}

export const Line = memo((props: LineProps): ReactElement | null => {
  const { setState } = Aether.use({
    type: LineGL.TYPE,
    schema: lineState,
    initialState: props,
  });
  useEffect(() => setState(props), [props]);
  return null;
});
Line.displayName = "Line";
