// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/lyra/flex";
import { type ReactElement } from "react";

import { Form as Base } from "@synnaxlabs/lyra/form";
import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Orientation } from "@/schematic/node/common/orientation";
import { type FormProps as NodeFormProps } from "@/schematic/node/spec";

export interface TankFormProps extends NodeFormProps {
  showBorderRadius?: boolean;
  showStrokeWidth?: boolean;
}

export const TankForm = ({
  showBorderRadius = false,
  showStrokeWidth = false,
}: TankFormProps): ReactElement => (
  <Form.Wrapper x align="stretch">
    <Flex.Box y grow>
      <Label.Form path="label" />
      <Flex.Box x>
        <Form.ColorField path="color" />
        <Form.ColorField path="backgroundColor" label="Background Color" />
        <Base.NumericField
          path="borderRadius.x"
          hideIfNull
          optional
          label="X Border Radius"
          grow
          inputProps={Form.PERCENT_BORDER_RADIUS_INPUT_PROPS}
        />
        <Base.NumericField
          path="borderRadius.y"
          hideIfNull
          optional
          label="Y Border Radius"
          grow
          inputProps={Form.PERCENT_BORDER_RADIUS_INPUT_PROPS}
        />
        {showBorderRadius && (
          <Base.NumericField
            path="borderRadius"
            hideIfNull
            optional
            label="Border Radius"
            grow
            inputProps={Form.DIMENSIONS_INPUT_PROPS}
          />
        )}
        {showStrokeWidth && (
          <Base.NumericField
            path="strokeWidth"
            hideIfNull
            optional
            label="Border Width"
            grow
            inputProps={Form.STROKE_WIDTH_INPUT_PROPS}
          />
        )}
        <Base.NumericField
          path="dimensions.width"
          label="Width"
          grow
          inputProps={Form.DIMENSIONS_INPUT_PROPS}
        />
        <Base.NumericField
          path="dimensions.height"
          label="Height"
          grow
          inputProps={Form.DIMENSIONS_INPUT_PROPS}
        />
      </Flex.Box>
    </Flex.Box>
    <Orientation.Field path="" hideInner showOuterCenter label="Label Location" />
  </Form.Wrapper>
);
