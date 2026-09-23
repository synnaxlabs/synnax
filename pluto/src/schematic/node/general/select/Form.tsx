// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type schematic } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { Channel } from "@/channel";
import { Form as Base } from "@/form";
import { Input } from "@/input";
import { Control } from "@/schematic/node/common/control";
import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Tabs } from "@/tabs";

const SelectTelemForm = ({ path }: { path: string }): ReactElement => {
  const { value, onChange } =
    Base.useField<
      Pick<schematic.SelectNodeConfig, "commandChannel" | "control" | "disabled">
    >(path);

  const handleSinkChange = (v: channel.Key): void =>
    onChange({
      ...value,
      commandChannel: v,
      control: Control.reveal(value.control),
      disabled: v === 0,
    });

  return (
    <Base.Sections x>
      <Base.Section title="Command">
        <Input.Item label="Channel" padHelpText={false}>
          <Channel.SelectSingle
            value={value.commandChannel ?? 0}
            onChange={handleSinkChange}
          />
        </Input.Item>
        <Form.ActivationDelayField />
        <Form.ControlChipField />
      </Base.Section>
    </Base.Sections>
  );
};

export const SelectForm = (): ReactElement => (
  <Form.Tabs tabs={["style", "control", "options"]}>
    <Tabs.Content itemKey="style">
      <Base.Sections x>
        <Base.Section title="Label">
          <Label.Form path="label" />
        </Base.Section>
        <Base.Section title="Appearance">
          <Form.ColorField path="color" />
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
  </Form.Tabs>
);
