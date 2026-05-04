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
import {
  ColorControl,
  DIMENSIONS_INPUT_PROPS,
  FormWrapper,
  LabelControls,
  OrientationControl,
} from "@/schematic/node/common/forms";

export const CylinderForm = (): ReactElement => (
  <FormWrapper x align="stretch">
    <Flex.Box y grow>
      <LabelControls path="label" />
      <Flex.Box x>
        <ColorControl path="color" />
        <ColorControl path="backgroundColor" label="Background Color" />
        <Form.NumericField
          path="dimensions.width"
          label="Width"
          grow
          defaultValue={200}
          inputProps={DIMENSIONS_INPUT_PROPS}
        />
        <Form.NumericField
          path="dimensions.height"
          label="Height"
          grow
          defaultValue={200}
          inputProps={DIMENSIONS_INPUT_PROPS}
        />
      </Flex.Box>
    </Flex.Box>
    <OrientationControl path="" hideInner />
  </FormWrapper>
);
