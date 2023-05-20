// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { XY } from "@synnaxlabs/x";
import { Dimensions } from "reactflow";

export const Canvas = {
  translate: (mat: DOMMatrix, xy: XY): DOMMatrix => mat.translate(xy.x, xy.y),
  scale: (mat: DOMMatrix, dims: Dimensions): DOMMatrix =>
    mat.scale(dims.width, dims.height),
};
