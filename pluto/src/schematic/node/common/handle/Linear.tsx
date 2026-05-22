// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type location } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Boundary } from "@/schematic/node/common/handle/Boundary";
import { Handle } from "@/schematic/node/common/handle/Handle";

export interface LinearProps {
  orientation: location.Outer;
  /// left is the horizontal position (%) of the left handle (id "1").
  left: number;
  /// right is the horizontal position (%) of the right handle (id "2").
  right: number;
  /// top is the shared vertical position (%) of both handles. Defaults to 50
  /// (vertically centered); set it for symbols whose pass-through line is offset.
  top?: number;
}

/// Linear renders the two opposing edge handles common to in-line, pass-through
/// symbols: one on the left edge and one on the right, sharing a vertical position.
/// Handle ids are fixed by side — left "1", right "2" — so connections stay
/// consistent across every symbol that uses it.
export const Linear = ({
  orientation,
  left,
  right,
  top = 50,
}: LinearProps): ReactElement => (
  <Boundary orientation={orientation}>
    <Handle location="left" orientation={orientation} left={left} top={top} id="1" />
    <Handle location="right" orientation={orientation} left={right} top={top} id="2" />
  </Boundary>
);
