// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/edge/jacketed/Jacketed.css";

import { type schematic } from "@synnaxlabs/client";
import { xy } from "@synnaxlabs/x";

import { CSS } from "@/css";
import { Base } from "@/schematic/edge/common/base";
import { Path } from "@/schematic/edge/common/path";
import { Segmented } from "@/schematic/edge/common/segmented";
import { type Spec } from "@/schematic/edge/spec";

const JACKET_GAP = 4;
const JACKET_CLASS = CSS.B("schematic-edge-jacket");

export const spec: Spec<"jacketed", schematic.JacketedEdgeConfig> =
  Segmented.createSpec(
    "jacketed",
    "Jacketed",
    ({ points, crossings, color, strokeWidth }) => {
      const miters = xy.calculateMiters(points, JACKET_GAP + strokeWidth);
      const above = points.map((p, i) => xy.translate(p, miters[i]));
      const below = points.map((p, i) => xy.translate(p, xy.scale(miters[i], -1)));
      return (
        <>
          <Base.Base
            path={Path.rounded(above, crossings)}
            color={color}
            strokeWidth={strokeWidth}
            className={JACKET_CLASS}
          />
          <Base.Base
            path={Path.rounded(points, crossings)}
            color={color}
            strokeWidth={strokeWidth}
          />
          <Base.Base
            path={Path.rounded(below, crossings)}
            color={color}
            strokeWidth={strokeWidth}
            className={JACKET_CLASS}
          />
        </>
      );
    },
  );
