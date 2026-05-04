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
import { Form } from "@/form";
import { type Input } from "@/input";
import {
  ColorControl,
  FormWrapper,
  LabelControls,
  STROKE_WIDTH_INPUT_PROPS,
} from "@/schematic/node/common/forms";

const RADIUS_INPUT_PROPS: Partial<Input.NumericProps> = {
  dragScale: { x: 0.5, y: 2.5 },
  bounds: { lower: 0, upper: 500 },
  endContent: "px",
};

export const CircleForm = (): ReactElement => (
  <FormWrapper direction="x" align="stretch">
    <Flex.Box direction="y" grow>
      <LabelControls path="label" />
      <Flex.Box direction="x">
        <ColorControl path="color" />
        <ColorControl path="backgroundColor" label="Background Color" />
        <Form.NumericField
          path="radius"
          label="Radius"
          inputProps={RADIUS_INPUT_PROPS}
          grow
        />
        <Form.NumericField
          path="strokeWidth"
          label="Border Width"
          inputProps={STROKE_WIDTH_INPUT_PROPS}
          grow
        />
      </Flex.Box>
    </Flex.Box>
  </FormWrapper>
);
