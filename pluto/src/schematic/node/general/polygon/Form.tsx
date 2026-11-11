// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { Form as Base } from "@synnaxlabs/lyra/form";
import { type Input } from "@synnaxlabs/lyra/input";
import { deep } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { fillFallback } from "@/schematic/node/general/polygon/Primitive";
import { type ColorFallbacks } from "@/schematic/node/spec";

const path = deep.path<schematic.PolygonNodeConfig>();

export const colorFallbacks = { fillColor: fillFallback } satisfies ColorFallbacks;

const ROTATION_INPUT_PROPS: Partial<Input.NumericProps> = {
  dragScale: { x: 0.5, y: 2 },
  bounds: { lower: 0, upper: 360 },
  endContent: "°",
};

const NUM_SIDES_INPUT_PROPS: Partial<Input.NumericProps> = {
  dragScale: { x: 0.02, y: 0.1 },
  bounds: { lower: 3, upper: 21 },
  integer: true,
};

const SIDE_LENGTH_INPUT_PROPS: Partial<Input.NumericProps> = {
  dragScale: { x: 0.5, y: 2.5 },
  bounds: { lower: 10, upper: 500 },
  endContent: "px",
  integer: true,
};

const BORDER_RADIUS_INPUT_PROPS: Partial<Input.NumericProps> = {
  dragScale: { x: 0.2, y: 1 },
  bounds: { lower: 0, upper: 181 },
  endContent: "px",
  integer: true,
};

export const PolygonForm = (): ReactElement => (
  <Base.Sections x>
    <Base.Section title="Label">
      <Label.Form path={path.label} />
    </Base.Section>
    <Base.Section title="Shape">
      <Base.NumericField
        path={path.numSides}
        label="Sides"
        inputProps={NUM_SIDES_INPUT_PROPS}
      />
      <Base.NumericField
        path={path.sideLength}
        label="Side length"
        inputProps={SIDE_LENGTH_INPUT_PROPS}
      />
      <Base.NumericField
        path={path.rotation}
        label="Rotation"
        inputProps={ROTATION_INPUT_PROPS}
      />
      <Base.NumericField
        path={path.borderRadius}
        label="Border radius"
        inputProps={BORDER_RADIUS_INPUT_PROPS}
      />
    </Base.Section>
    <Base.Section title="Appearance">
      <Form.ColorField path={path.strokeColor} label="Stroke" />
      <Form.FillField fallback={colorFallbacks.fillColor} />
      <Base.NumericField
        path={path.strokeWidth}
        label="Border width"
        inputProps={Form.STROKE_WIDTH_INPUT_PROPS}
      />
    </Base.Section>
  </Base.Sections>
);
