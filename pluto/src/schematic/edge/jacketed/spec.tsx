// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { xy } from "@synnaxlabs/x";

import { Base } from "@/schematic/edge/common/base";
import { Path } from "@/schematic/edge/common/path";
import { Segmented } from "@/schematic/edge/common/segmented";
import { type Spec } from "@/schematic/edge/spec";

const JACKET_OFFSET = 6;
const JACKET_OPACITY = 0.7;
const JACKET_STYLE = { opacity: JACKET_OPACITY };

export const spec: Spec<"jacketed", schematic.EdgeConfigJacketed> =
  Segmented.createSpec("jacketed", "Jacketed", ({ points, crossings, color }) => {
    const miters = xy.calculateMiters(points, JACKET_OFFSET);
    const above = points.map((p, i) => xy.translate(p, miters[i]));
    const below = points.map((p, i) => xy.translate(p, xy.scale(miters[i], -1)));
    return (
      <>
        <Base.Base
          path={Path.rounded(above, crossings)}
          color={color}
          style={JACKET_STYLE}
        />
        <Base.Base path={Path.rounded(points, crossings)} color={color} />
        <Base.Base
          path={Path.rounded(below, crossings)}
          color={color}
          style={JACKET_STYLE}
        />
      </>
    );
  });
