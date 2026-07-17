// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type text } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Flex } from "@/flex";
import { Form as Base } from "@/form";
import { type Input } from "@/input";
import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Select } from "@/select";
import { Tabs } from "@/tabs";
import { Value } from "@/vis/value";

const GAUGE_BAR_WIDTH_INPUT_PROPS: Partial<Input.NumericProps> = {
  min: 1,
  max: 50,
  step: 1,
  bounds: { lower: 1, upper: 50 },
  endContent: "px",
  dragScale: { x: 0.1, y: 0.1 },
};

const BOUND_INPUT_PROPS: Partial<Input.NumericProps> = { step: 10 };

const handleLevelChange = (v: text.Level, { set }: Base.ContextValue): void => {
  if (v === "small") set("barWidth", 4);
  else if (v === "h5") set("barWidth", 8);
  else set("barWidth", 10);
};

export const GaugeForm = (): ReactElement => (
  <Tabs.Frame initialValue="properties">
    <Tabs.Selector>
      <Tabs.Tab itemKey="properties">Properties</Tabs.Tab>
      <Tabs.Tab itemKey="telemetry">Telemetry</Tabs.Tab>
    </Tabs.Selector>
    <Tabs.Content itemKey="properties">
      <Form.Wrapper x>
        <Flex.Box y grow>
          <Label.Form path="label" />
          <Flex.Box x>
            <Form.ColorField path="color" />
            <Form.UnitsField />
            <Base.NumericField
              path="bounds.lower"
              label="Min value"
              hideIfNull
              inputProps={BOUND_INPUT_PROPS}
            />
            <Base.NumericField
              path="bounds.upper"
              label="Max value"
              hideIfNull
              inputProps={BOUND_INPUT_PROPS}
            />
            <Base.NumericField
              path="barWidth"
              label="Bar width"
              hideIfNull
              inputProps={GAUGE_BAR_WIDTH_INPUT_PROPS}
            />
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
          </Flex.Box>
        </Flex.Box>
      </Form.Wrapper>
    </Tabs.Content>
    <Tabs.Content itemKey="telemetry">
      <Form.Wrapper y empty>
        <Value.TelemForm path="" />
      </Form.Wrapper>
    </Tabs.Content>
  </Tabs.Frame>
);
