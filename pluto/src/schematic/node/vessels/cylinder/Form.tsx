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
import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Orientation } from "@/schematic/node/common/orientation";
export const CylinderForm = (): ReactElement => (
  <Form.Wrapper x align="stretch">
    <Flex.Box y grow>
      <Label.Form path="label" />
      <Flex.Box x>
        <Form.ColorField path="color" />
        <Form.ColorField path="backgroundColor" label="Background color" />
        <Base.NumericField
          path="dimensions.width"
          label="Width"
          grow
          defaultValue={200}
          inputProps={Form.DIMENSIONS_INPUT_PROPS}
        />
        <Base.NumericField
          path="dimensions.height"
          label="Height"
          grow
          defaultValue={200}
          inputProps={Form.DIMENSIONS_INPUT_PROPS}
        />
      </Flex.Box>
    </Flex.Box>
    <Orientation.Field path="" />
  </Form.Wrapper>
);
