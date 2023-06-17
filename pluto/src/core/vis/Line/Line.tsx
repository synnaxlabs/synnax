// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, memo } from "react";

import { Optional } from "@synnaxlabs/x";

import { Aether } from "@/core/aether/main";
import { LineState } from "@/core/vis/Line/core";
import { LineGL } from "@/core/vis/Line/LineGL";

export interface LineProps extends Optional<Omit<LineState, "key">, "strokeWidth"> {}

export const Line = memo((props: LineProps): ReactElement | null => {
  Aether.use<LineState>(LineGL.TYPE, props);
  return null;
});
Line.displayName = "Line";
