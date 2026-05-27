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

export interface RectangleProps {
  orientation: location.Outer;
  /// left is the horizontal position (%) of the left-edge handle (id "1").
  left: number;
  /// top is the vertical position (%) of the top-edge handle (id "3").
  top: number;
  /// right is the horizontal position (%) of the right-edge handle (id "2").
  right: number;
  /// bottom is the vertical position (%) of the bottom-edge handle (id "4").
  bottom: number;
  /// refreshDeps, when changed, forces the handles to recompute their positions
  /// (forwarded to the underlying Boundary).
  refreshDeps?: unknown;
}

/// Rectangle renders the four edge handles common to box-shaped symbols: one per side,
/// centered on the perpendicular axis. Handle ids are fixed by side — left "1", right
/// "2", top "3", bottom "4" — so connections stay consistent across every symbol that
/// uses it.
export const Rectangle = ({
  orientation,
  left,
  top,
  right,
  bottom,
  refreshDeps,
}: RectangleProps): ReactElement => (
  <Boundary orientation={orientation} refreshDeps={refreshDeps}>
    <Handle location="left" orientation={orientation} left={left} top={50} id="1" />
    <Handle location="right" orientation={orientation} left={right} top={50} id="2" />
    <Handle location="top" orientation={orientation} left={50} top={top} id="3" />
    <Handle location="bottom" orientation={orientation} left={50} top={bottom} id="4" />
  </Boundary>
);
