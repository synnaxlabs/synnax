// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { location } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Form as Base } from "@/form";
import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Orientation } from "@/schematic/node/common/orientation";
import { Scale } from "@/schematic/node/common/scale";
import { Tabs } from "@/tabs";

export const ScaleForm = (): ReactElement => {
  const { value: orientation } = Base.useField<location.Outer>("orientation");
  return (
    <Form.Tabs tabs={["style", "telemetry"]}>
      <Tabs.Content itemKey="style">
        <Base.Sections x>
          <Base.Section title="Label">
            <Label.Form path="label" />
          </Base.Section>
          <Base.Section title="Dimensions">
            <Base.NumericField
              path="dimensions.width"
              label="Width"
              padHelpText={false}
              inputProps={Form.DIMENSIONS_INPUT_PROPS}
            />
            <Base.NumericField
              path="dimensions.height"
              label="Height"
              padHelpText={false}
              inputProps={Form.DIMENSIONS_INPUT_PROPS}
            />
          </Base.Section>
          <Base.Section title="Display">
            <Scale.DisplayFields
              path="indicator"
              axis={location.direction(orientation)}
            />
          </Base.Section>
          <Base.Section title="Appearance">
            <Form.ColorField path="color" label="Fill color" />
            <Scale.StyleFields path="indicator" />
          </Base.Section>
          <Orientation.Section path="" hideInner />
        </Base.Sections>
      </Tabs.Content>
      <Tabs.Content itemKey="telemetry">
        <Base.Sections x>
          <Scale.TelemForm path="indicator" />
        </Base.Sections>
      </Tabs.Content>
    </Form.Tabs>
  );
};
