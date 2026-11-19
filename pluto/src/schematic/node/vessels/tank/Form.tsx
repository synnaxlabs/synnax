// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Form as Base } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Tabs } from "@synnaxlabs/lyra/tabs";
import { deep, primitive } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Border } from "@/schematic/node/common/border";
import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Orientation } from "@/schematic/node/common/orientation";
import { Scale } from "@/schematic/node/common/scale";
import { type FormProps as NodeFormProps } from "@/schematic/node/spec";
import { Properties } from "@/vis/properties";

const path = deep.path<schematic.TankNodeConfig>();
const boxPath = deep.path<schematic.BoxNodeConfig>();

const BoxBorderRadiusNumeric = ({
  value,
  onChange,
}: Input.Control<number | undefined>): ReactElement => (
  <Input.Numeric
    value={value ?? Border.DEFAULT_BOX_RADIUS}
    onChange={onChange}
    {...Form.DIMENSIONS_INPUT_PROPS}
    integer
  />
);

const BoxBorderRadiusInput = Component.renderProp(BoxBorderRadiusNumeric);

export interface TankFormProps extends NodeFormProps {
  showBorderRadius?: boolean;
  showStrokeWidth?: boolean;
  showFillTab?: boolean;
}

const FillForm = (): ReactElement => {
  const channel = Base.useFieldValue(path.channel);
  return (
    <Base.Sections x>
      <Scale.TelemForm allowNone />
      {primitive.isNonZero(channel) && (
        <>
          <Base.Section title="Display">
            <Scale.DisplayFields>
              <Base.SwitchField
                path={path.caretVisible}
                label="Value"
                padHelpText={false}
              />
              <Base.SwitchField
                path={path.scaleVisible}
                label="Scale"
                padHelpText={false}
              />
            </Scale.DisplayFields>
          </Base.Section>
          <Base.Section title="Appearance">
            <Scale.StyleFields />
          </Base.Section>
        </>
      )}
    </Base.Sections>
  );
};

export const TankForm = ({
  tab,
  onTabChange,
  showBorderRadius = false,
  showStrokeWidth = false,
  showFillTab = false,
}: TankFormProps): ReactElement => {
  const style = (
    <Base.Sections x>
      <Base.Section title="Label">
        <Label.Form path={path.label} />
        <Orientation.Field path={path} hideInner showOuterCenter label="Location" />
      </Base.Section>
      <Base.Section title="Appearance">
        <Form.ColorField path={path.strokeColor} label="Stroke" />
        <Form.FillField />
        {showBorderRadius ? (
          <Base.Field path={boxPath.borderRadius} label="Border radius">
            {BoxBorderRadiusInput}
          </Base.Field>
        ) : (
          <Form.RadiusFields path={path.borderRadius} />
        )}
        {showStrokeWidth && <Form.StrokeWidthField />}
      </Base.Section>
      <Base.Section title="Dimensions">
        <Base.NumericField
          path={path.dimensions.width}
          label="Width"
          inputProps={Form.DIMENSIONS_INPUT_PROPS}
        />
        <Base.NumericField
          path={path.dimensions.height}
          label="Height"
          inputProps={Form.DIMENSIONS_INPUT_PROPS}
        />
      </Base.Section>
    </Base.Sections>
  );
  if (!showFillTab) return style;
  return (
    <Properties.Tabs tabs={["style", "fill"]} tab={tab} onTabChange={onTabChange}>
      <Tabs.Content itemKey="style">{style}</Tabs.Content>
      <Tabs.Content itemKey="fill">
        <FillForm />
      </Tabs.Content>
    </Properties.Tabs>
  );
};
