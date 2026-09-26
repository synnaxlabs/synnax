// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Form as Base } from "@/form";
import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Orientation } from "@/schematic/node/common/orientation";
import { Scale } from "@/schematic/node/common/scale";
import { type FormProps as NodeFormProps } from "@/schematic/node/spec";
import { Tabs } from "@/tabs";

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
    <Base.Sections x>
      <Scale.TelemForm path="fill" allowNone />
      {channel != null && (
        <>
          <Base.Section title="Display">
            <Scale.DisplayFields path="fill" />
          </Base.Section>
          <Base.Section title="Appearance">
            <Form.ColorField path="fill.color" label="Fill color" />
            <Scale.StyleFields path="fill" />
          </Base.Section>
        </>
      )}
    </Base.Sections>
  );
};

export const TankForm = ({
  showBorderRadius = false,
  showStrokeWidth = false,
  showFillTab = false,
}: TankFormProps): ReactElement => {
  const style = (
    <Base.Sections x>
      <Base.Section title="Label">
        <Label.Form path="label" />
        <Orientation.Field path="" hideInner showOuterCenter label="Location" />
      </Base.Section>
      <Base.Section title="Appearance">
        <Form.ColorField path="color" />
        <Form.ColorField path="backgroundColor" label="Background color" />
        <Form.RadiusFields path="borderRadius" />
        {showBorderRadius && (
          <Base.NumericField
            path="borderRadius"
            hideIfNull
            optional
            label="Border radius"
            inputProps={Form.DIMENSIONS_INPUT_PROPS}
          />
        )}
        {showStrokeWidth && (
          <Base.NumericField
            path="strokeWidth"
            hideIfNull
            optional
            label="Border width"
            inputProps={Form.STROKE_WIDTH_INPUT_PROPS}
          />
        )}
      </Base.Section>
      <Base.Section title="Dimensions">
        <Base.NumericField
          path="dimensions.width"
          label="Width"
          inputProps={Form.DIMENSIONS_INPUT_PROPS}
        />
        <Base.NumericField
          path="dimensions.height"
          label="Height"
          inputProps={Form.DIMENSIONS_INPUT_PROPS}
        />
      </Base.Section>
    </Base.Sections>
  );
  if (!showFillTab) return style;
  return (
    <Form.Tabs tabs={["style", "fill"]}>
      <Tabs.Content itemKey="style">{style}</Tabs.Content>
      <Tabs.Content itemKey="fill">
        <FillForm />
      </Tabs.Content>
    </Form.Tabs>
  );
};
