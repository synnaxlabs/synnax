// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form as Base } from "@synnaxlabs/lyra/form";
import { type Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Tabs } from "@synnaxlabs/lyra/tabs";
import { type text } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Value } from "@/vis/value";

const GAUGE_BAR_WIDTH_INPUT_PROPS: Partial<Input.NumericProps> = {
  min: 1,
  max: 50,
  step: 1,
  bounds: { lower: 1, upper: 50 },
  endContent: "px",
  dragScale: { x: 0.1, y: 0.1 },
};

const handleLevelChange = (v: text.Level, { set }: Base.ContextValue): void => {
  if (v === "small") set("barWidth", 4);
  else if (v === "h5") set("barWidth", 8);
  else set("barWidth", 10);
};

export const GaugeForm = (): ReactElement => (
  <Form.Tabs tabs={["style", "telemetry"]}>
    <Tabs.Content itemKey="style">
      <Base.Sections x>
        <Base.Section title="Label">
          <Label.Form path="label" />
        </Base.Section>
        <Base.Section title="Appearance">
          <Form.ColorField path="color" />
          <Base.Field<text.Level>
            path="level"
            label="Size"
            hideIfNull
            padHelpText={false}
            onChange={handleLevelChange}
          >
            {({ value, onChange }) => (
              <Select.Text.Level value={value} onChange={onChange} />
            )}
          </Base.Field>
          <Base.NumericField
            path="barWidth"
            label="Bar width"
            hideIfNull
            padHelpText={false}
            inputProps={GAUGE_BAR_WIDTH_INPUT_PROPS}
          />
        </Base.Section>
        <Base.Section title="Range">
          <Form.UnitsField />
          <Form.BoundsFields path="bounds" hideIfNull padHelpText={false} />
        </Base.Section>
      </Base.Sections>
    </Tabs.Content>
    <Tabs.Content itemKey="telemetry">
      <Base.Sections x>
        <Value.TelemForm path="" />
      </Base.Sections>
    </Tabs.Content>
  </Form.Tabs>
);
