// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/edge/electric/Electric.css";

import { CSS } from "@/css";
import { Base } from "@/schematic/edge/common/base";
import { Path } from "@/schematic/edge/common/path";
import { Segmented } from "@/schematic/edge/common/segmented";
import { NAME, VARIANT } from "@/schematic/edge/electric/config";

export const spec = Segmented.createSpec(
  VARIANT,
  NAME,
  ({ points, crossings, color }) => (
    <Base.Base
      path={Path.rounded(points, crossings)}
      color={color}
      className={CSS.B("schematic-edge-electric")}
    />
  ),
);
