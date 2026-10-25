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
import { type ReactElement } from "react";

import { Channel } from "@/channel";
import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { type ColorFallbacks, type FormProps } from "@/schematic/node/spec";
import { Properties } from "@/vis/properties";
import { Staleness } from "@/vis/staleness";

export const colorFallbacks = {
  fillColor: Form.primaryFallback,
} satisfies ColorFallbacks;

const SelectTelemForm = ({ path }: { path: string }): ReactElement => {
  const { value, onChange } =
    Base.useField<
      Pick<schematic.SelectNodeConfig, "commandChannel" | "control" | "disabled">
    >(path);

  const handleSinkChange = (v: channel.Key | null): void => {
    const commandChannel = v ?? 0;
    onChange({
      ...value,
      commandChannel,
      disabled: commandChannel === 0,
    });
  };

  return (
    <Base.Sections x>
      <Form.StateSection />
      <Base.Section title="Command">
        <Input.Item label="Channel" padHelpText={false}>
          <Channel.SelectSingle
            allowNone
            value={value.commandChannel ?? 0}
            onChange={handleSinkChange}
          />
        </Input.Item>
        <Form.ActivationDelayField />
        <Form.ControlChipField />
      </Base.Section>
      <Base.Section title="Staleness">
        <Staleness.Fields />
      </Base.Section>
    </Base.Sections>
  );
};

export const SelectForm = ({ tab, onTabChange }: FormProps): ReactElement => (
  <Properties.Tabs
    tabs={["control", "style", "options"]}
    tab={tab}
    onTabChange={onTabChange}
  >
    <Tabs.Content itemKey="style">
      <Base.Sections x>
        <Base.Section title="Label">
          <Label.Form path="label" />
        </Base.Section>
        <Base.Section title="Appearance">
          <Form.FillField fallback={colorFallbacks.fillColor} />
          <Form.SizeField />
          <Base.NumericField
            path="inlineSize"
            label="Width"
            padHelpText={false}
            inputProps={Form.VALUE_WIDTH_INPUT_PROPS}
          />
        </Base.Section>
      </Base.Sections>
    </Tabs.Content>
    <Tabs.Content itemKey="control">
      <SelectTelemForm path="" />
    </Tabs.Content>
    <Tabs.Content itemKey="options">
      <Form.StateMappingForm path="options" />
    </Tabs.Content>
  </Properties.Tabs>
);
