// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { color } from "@synnaxlabs/x";

import { removeProps } from "@/component/removeProps";
import { Label } from "@/schematic/node/common/label";
import { CommonPolygonForm } from "@/schematic/node/general/polygon/Form";
import {
  DEFAULT_POLYGON_SIDE_LENGTH,
  Polygon,
} from "@/schematic/node/general/polygon/Primitive";
import { Symbol } from "@/schematic/node/general/polygon/Symbol";
import { type Spec } from "@/schematic/node/spec";
import { type Theming } from "@/theming";

export const defaultConfig = (t: Theming.Theme): schematic.NodeConfigPolygon => ({
  variant: "polygon",
  numSides: 6,
  sideLength: DEFAULT_POLYGON_SIDE_LENGTH,
  cornerRounding: 0,
  rotation: 0,
  color: color.ZERO,
  backgroundColor: color.setAlpha(t.colors.gray.l1, 0),
  strokeWidth: 2,
  label: Label.defaultConfig("Polygon"),
});

export const spec: Spec<"polygon", schematic.NodeConfigPolygon> = {
  key: "polygon",
  name: "Polygon",
  Form: CommonPolygonForm,
  Node: Symbol,
  Preview: removeProps(Polygon, ["clickable"]),
  defaultConfig,
  zIndex: 2,
};
