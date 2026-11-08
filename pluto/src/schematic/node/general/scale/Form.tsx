// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { Form as Base } from "@synnaxlabs/lyra/form";
import { Tabs } from "@synnaxlabs/lyra/tabs";
import { deep, location } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Orientation } from "@/schematic/node/common/orientation";
import { Scale } from "@/schematic/node/common/scale";
import { type FormProps } from "@/schematic/node/spec";
import { Properties } from "@/vis/properties";

const path = deep.path<schematic.ScaleNodeConfig>();

export const ScaleForm = ({ tab, onTabChange }: FormProps): ReactElement => {
  const { value: orientation } = Base.useField(path.orientation);
  return (
    <Properties.Tabs tabs={["telemetry", "style"]} tab={tab} onTabChange={onTabChange}>
      <Tabs.Content itemKey="style">
        <Base.Sections x>
          <Base.Section title="Label">
            <Label.Form path={path.label} />
          </Base.Section>
          <Base.Section title="Dimensions">
            <Base.NumericField
              path={path.dimensions.width}
              label="Width"
              padHelpText={false}
              inputProps={Form.DIMENSIONS_INPUT_PROPS}
            />
            <Base.NumericField
              path={path.dimensions.height}
              label="Height"
              padHelpText={false}
              inputProps={Form.DIMENSIONS_INPUT_PROPS}
            />
          </Base.Section>
          <Base.Section title="Display">
            <Scale.DisplayFields axis={location.direction(orientation)}>
              <Form.NegatedSwitchField
                path={path.caretHidden}
                label="Value"
                padHelpText={false}
              />
              <Form.NegatedSwitchField
                path={path.scaleHidden}
                label="Scale"
                padHelpText={false}
              />
            </Scale.DisplayFields>
          </Base.Section>
          <Base.Section title="Appearance">
            <Scale.StyleFields />
          </Base.Section>
          <Orientation.Section path={path} hideInner />
        </Base.Sections>
      </Tabs.Content>
      <Tabs.Content itemKey="telemetry">
        <Base.Sections x>
          <Scale.TelemForm />
        </Base.Sections>
      </Tabs.Content>
    </Properties.Tabs>
  );
};
