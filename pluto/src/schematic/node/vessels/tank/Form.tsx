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
import { Scale } from "@/schematic/node/common/scale";
import { type FormProps as NodeFormProps } from "@/schematic/node/spec";
import { Tabs } from "@/tabs";

/** Fill defaults for a tank, which draws its own wall and reads best with no scale. */
export const FILL_DEFAULTS: Partial<Scale.Config> = {
  showCaret: false,
  showScale: false,
  side: "left",
};

export interface TankFormProps extends NodeFormProps {
  showBorderRadius?: boolean;
  showStrokeWidth?: boolean;
  showFillTab?: boolean;
}

const FillForm = (): ReactElement => {
  const channel = Base.useFieldValue<Scale.Config["channel"]>("fill.channel", {
    optional: true,
  });
  return (
    <Form.Wrapper y empty>
      <Scale.TelemForm path="fill" defaults={FILL_DEFAULTS} allowNone />
      {channel != null && (
        <Flex.Box x>
          <Scale.DisplayFields path="fill" />
          <Form.ColorField path="fill.color" label="Fill color" />
          <Scale.StyleFields path="fill" />
        </Flex.Box>
      )}
    </Form.Wrapper>
  );
};

export const TankForm = ({
  showBorderRadius = false,
  showStrokeWidth = false,
  showFillTab = false,
}: TankFormProps): ReactElement => {
  const properties = (
    <Form.Wrapper x align="stretch">
      <Flex.Box y grow>
        <Label.Form path="label" />
        <Flex.Box x>
          <Form.ColorField path="color" />
          <Form.ColorField path="backgroundColor" label="Background color" />
          <Form.RadiusFields path="borderRadius" />
          {showBorderRadius && (
            <Base.NumericField
              path="borderRadius"
              hideIfNull
              optional
              label="Border radius"
              grow
              inputProps={Form.DIMENSIONS_INPUT_PROPS}
            />
          )}
          {showStrokeWidth && (
            <Base.NumericField
              path="strokeWidth"
              hideIfNull
              optional
              label="Border width"
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
      <Orientation.Field path="" hideInner showOuterCenter label="Label location" />
    </Form.Wrapper>
  );
  if (!showFillTab) return properties;
  return (
    <Tabs.Frame initialValue="properties">
      <Tabs.Selector>
        <Tabs.Tab itemKey="properties">Properties</Tabs.Tab>
        <Tabs.Tab itemKey="fill">Fill</Tabs.Tab>
      </Tabs.Selector>
      <Tabs.Content itemKey="properties">{properties}</Tabs.Content>
      <Tabs.Content itemKey="fill">
        <FillForm />
      </Tabs.Content>
    </Tabs.Frame>
  );
};
