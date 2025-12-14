// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type optional } from "@synnaxlabs/x";
import { memo, type ReactElement } from "react";

import { Aether } from "@/aether";
import { line } from "@/vis/line/aether";

export interface LineProps
  extends
    optional.Optional<Omit<line.State, "key">, "strokeWidth">,
    Aether.ComponentProps {}

export const Line = memo(({ aetherKey, ...rest }: LineProps): ReactElement | null => {
  Aether.useUnidirectional({
    aetherKey,
    type: line.Line.TYPE,
    schema: line.stateZ,
    state: rest,
  });
  return null;
});
Line.displayName = "Line";
