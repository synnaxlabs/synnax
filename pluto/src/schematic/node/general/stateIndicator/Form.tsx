// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type schematic } from "@synnaxlabs/client";
import { Form as Base } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Tabs } from "@synnaxlabs/lyra/tabs";
import { deep } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Channel } from "@/channel";
import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { type FormProps } from "@/schematic/node/spec";
import { Properties } from "@/vis/properties";
import { Staleness } from "@/vis/staleness";

const path = deep.path<schematic.StateIndicatorNodeConfig>();

const StateIndicatorTelemForm = (): ReactElement => {
  const { value, onChange } = Base.useField(path);

  const handleSourceChange = (v: channel.Key | null): void =>
    onChange({ ...value, channel: v ?? undefined });

  return (
    <Base.Sections x>
      <Base.Section title="State">
        <Input.Item label="Channel" padHelpText={false}>
          <Channel.SelectSingle
            allowNone
            value={value.channel ?? 0}
            onChange={handleSourceChange}
          />
        </Input.Item>
      </Base.Section>
      <Base.Section title="Staleness">
        <Staleness.Fields />
      </Base.Section>
    </Base.Sections>
  );
};

export const StateIndicatorForm = ({ tab, onTabChange }: FormProps): ReactElement => (
  <Properties.Tabs
    tabs={["telemetry", "style", "options"]}
    tab={tab}
    onTabChange={onTabChange}
  >
    <Tabs.Content itemKey="style">
      <Base.Sections x>
        <Base.Section title="Label">
          <Label.Form path={path.label} />
        </Base.Section>
        <Base.Section title="Appearance">
          <Form.ColorField path={path.strokeColor} label="Stroke" />
          <Form.ScaledSizeField />
          <Base.NumericField
            path={path.inlineSize}
            label="Width"
            padHelpText={false}
            inputProps={Form.VALUE_WIDTH_INPUT_PROPS}
          />
        </Base.Section>
      </Base.Sections>
    </Tabs.Content>
    <Tabs.Content itemKey="telemetry">
      <StateIndicatorTelemForm />
    </Tabs.Content>
    <Tabs.Content itemKey="options">
      <Form.StateMappingForm path={path.options} showColor />
    </Tabs.Content>
  </Properties.Tabs>
);
