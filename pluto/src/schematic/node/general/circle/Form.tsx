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
const RADIUS_INPUT_PROPS: Partial<Input.NumericProps> = {
  dragScale: { x: 0.5, y: 2.5 },
  bounds: { lower: 0, upper: 500 },
  endContent: "px",
};

export const CircleForm = (): ReactElement => (
  <Form.Wrapper direction="x" align="stretch">
    <Flex.Box direction="y" grow>
      <Label.Form path="label" />
      <Flex.Box direction="x">
        <Form.ColorField path="color" />
        <Form.ColorField path="backgroundColor" label="Background color" />
        <Base.NumericField
          path="radius"
          label="Radius"
          inputProps={RADIUS_INPUT_PROPS}
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
