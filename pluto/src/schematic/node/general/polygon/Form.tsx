// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Flex } from "@/flex";
import { Form as Base } from "@/form";
import { type Input } from "@/input";
import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
export interface PolygonFormProps {
  numSides: number;
}

const ROTATION_INPUT_PROPS: Partial<Input.NumericProps> = {
  dragScale: { x: 0.5, y: 2 },
  bounds: { lower: 0, upper: 360 },
  endContent: "°",
};

const NUM_SIDES_INPUT_PROPS: Partial<Input.NumericProps> = {
  dragScale: { x: 0.02, y: 0.1 },
  bounds: { lower: 3, upper: 21 },
};

const SIDE_LENGTH_INPUT_PROPS: Partial<Input.NumericProps> = {
  dragScale: { x: 0.5, y: 2.5 },
  bounds: { lower: 10, upper: 500 },
  endContent: "px",
};

const CORNER_ROUNDING_INPUT_PROPS: Partial<Input.NumericProps> = {
  dragScale: { x: 0.2, y: 1 },
  bounds: { lower: 0, upper: 181 },
  endContent: "px",
};

export const PolygonForm = (): ReactElement => (
  <Form.Wrapper direction="x" align="stretch">
    <Flex.Box direction="y" grow>
      <Label.Form path="label" />
      <Flex.Box direction="x">
        <Form.ColorField path="color" />
        <Form.ColorField path="backgroundColor" label="Background color" />
        <Base.NumericField
          path="rotation"
          label="Rotation"
          inputProps={ROTATION_INPUT_PROPS}
          grow
        />
        <Base.NumericField
          path="numSides"
          label="Number of sides"
          inputProps={NUM_SIDES_INPUT_PROPS}
          grow
        />
        <Base.NumericField
          path="sideLength"
          label="Side length"
          inputProps={SIDE_LENGTH_INPUT_PROPS}
          grow
        />
        <Base.NumericField
          path="cornerRounding"
          label="Corner rounding"
          inputProps={CORNER_ROUNDING_INPUT_PROPS}
          grow
        />
        <Base.NumericField
          path="strokeWidth"
          label="Border width"
          inputProps={Form.STROKE_WIDTH_INPUT_PROPS}
          grow
        />
      </Flex.Box>
    </Flex.Box>
  </Form.Wrapper>
);

export const CommonPolygonForm = PolygonForm;
