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
import {
  ColorControl,
  DIMENSIONS_INPUT_PROPS,
  FormWrapper,
  LabelControls,
  OrientationControl,
  PERCENT_BORDER_RADIUS_INPUT_PROPS,
  STROKE_WIDTH_INPUT_PROPS,
  type SymbolFormProps,
} from "@/schematic/node/common/forms";

export interface FormProps extends SymbolFormProps {
  showBorderRadius?: boolean;
  showStrokeWidth?: boolean;
}

export const Form = ({
  showBorderRadius = false,
  showStrokeWidth = false,
}: FormProps): ReactElement => (
  <FormWrapper x align="stretch">
    <Flex.Box y grow>
      <LabelControls path="label" />
      <Flex.Box x>
        <ColorControl path="color" />
        <ColorControl path="backgroundColor" label="Background Color" />
        <Base.NumericField
          path="borderRadius.x"
          hideIfNull
          optional
          label="X Border Radius"
          grow
          inputProps={PERCENT_BORDER_RADIUS_INPUT_PROPS}
        />
        <Base.NumericField
          path="borderRadius.y"
          hideIfNull
          optional
          label="Y Border Radius"
          grow
          inputProps={PERCENT_BORDER_RADIUS_INPUT_PROPS}
        />
        {showBorderRadius && (
          <Base.NumericField
            path="borderRadius"
            hideIfNull
            optional
            label="Border Radius"
            grow
            inputProps={DIMENSIONS_INPUT_PROPS}
          />
        )}
        {showStrokeWidth && (
          <Base.NumericField
            path="strokeWidth"
            hideIfNull
            optional
            label="Border Width"
            grow
            inputProps={STROKE_WIDTH_INPUT_PROPS}
          />
        )}
        <Base.NumericField
          path="dimensions.width"
          label="Width"
          grow
          inputProps={DIMENSIONS_INPUT_PROPS}
        />
        <Base.NumericField
          path="dimensions.height"
          label="Height"
          grow
          inputProps={DIMENSIONS_INPUT_PROPS}
        />
      </Flex.Box>
    </Flex.Box>
    <OrientationControl path="" hideInner showOuterCenter label="Label Location" />
  </FormWrapper>
);
