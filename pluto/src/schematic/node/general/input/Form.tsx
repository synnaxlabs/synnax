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
interface InputTelemFormProps {
  path: string;
}

const InputTelemForm = ({ path }: InputTelemFormProps): ReactElement => {
  const { value, onChange } =
    Base.useField<
      Pick<schematic.InputNodeConfig, "commandChannel" | "control" | "disabled">
    >(path);
  const handleSinkChange = (v: channel.Key | null): void => {
    v ??= 0;
    onChange({
      ...value,
      commandChannel: v,
      control: Control.reveal(value.control),
      disabled: v === 0,
    });
  };

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

export const InputForm = (): ReactElement => (
  <Form.Tabs tabs={["style", "control"]}>
    <Tabs.Content itemKey="style">
      <Base.Sections x>
        <Base.Section title="Label">
          <Label.Form path="label" />
        </Base.Section>
        <Base.Section title="Appearance">
          <Form.ColorField path="color" />
          <Form.SizeField />
        </Base.Section>
      </Base.Sections>
    </Tabs.Content>
    <Tabs.Content itemKey="control">
      <InputTelemForm path="" />
    </Tabs.Content>
  </Form.Tabs>
);
