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
import { type Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Tabs } from "@synnaxlabs/lyra/tabs";
import { deep, type text } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { type ColorFallbacks, type FormProps } from "@/schematic/node/spec";
import { Gauge as VisGauge } from "@/vis/gauge";
import { Properties } from "@/vis/properties";
import { Value } from "@/vis/value";

const path = deep.path<schematic.GaugeNodeConfig>();

export const colorFallbacks = {
  strokeColor: VisGauge.colorFallback,
} satisfies ColorFallbacks;

const GAUGE_BAR_WIDTH_INPUT_PROPS: Partial<Input.NumericProps> = {
  min: 1,
  max: 50,
  step: 1,
  bounds: { lower: 1, upper: 50 },
  endContent: "px",
  integer: true,
  dragScale: { x: 0.1, y: 0.1 },
};

const handleLevelChange = (v: text.Level, { set }: Base.ContextValue): void => {
  if (v === "small") set(path.barWidth, 4);
  else if (v === "h5") set(path.barWidth, 8);
  else set(path.barWidth, 10);
};

export const GaugeForm = ({ tab, onTabChange }: FormProps): ReactElement => (
  <Properties.Tabs tabs={["telemetry", "style"]} tab={tab} onTabChange={onTabChange}>
    <Tabs.Content itemKey="style">
      <Base.Sections x>
        <Base.Section title="Label">
          <Label.Form path={path.label} />
        </Base.Section>
        <Base.Section title="Appearance">
          <Form.ColorField
            path={path.strokeColor}
            label="Stroke"
            fallback={colorFallbacks.strokeColor}
          />
          <Base.Field
            path={path.level}
            label="Size"
            padHelpText={false}
            onChange={handleLevelChange}
          >
            {({ value, onChange }) => (
              <Select.Text.Level value={value} onChange={onChange} />
            )}
          </Base.Field>
          <Base.NumericField
            path={path.barWidth}
            label="Bar width"
            padHelpText={false}
            inputProps={GAUGE_BAR_WIDTH_INPUT_PROPS}
          />
        </Base.Section>
        <Base.Section title="Range">
          <Form.UnitsField />
          <Form.BoundsFields path={path.bounds} padHelpText={false} />
        </Base.Section>
      </Base.Sections>
    </Tabs.Content>
    <Tabs.Content itemKey="telemetry">
      <Base.Sections x>
        <Value.TelemForm />
      </Base.Sections>
    </Tabs.Content>
  </Properties.Tabs>
);
