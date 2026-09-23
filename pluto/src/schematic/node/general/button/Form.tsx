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
import { Button as BaseButton } from "@/vis/button";

type ButtonTelemFormT = Pick<
  schematic.ButtonNodeConfig,
  "commandChannel" | "control" | "mode"
>;

export const ButtonTelemForm = ({ path }: { path: string }): ReactElement => {
  const { value, onChange } = Base.useField<ButtonTelemFormT>(path);
  const handleSinkChange = (v: channel.Key): void =>
    onChange({
      ...value,
      commandChannel: v,
      control: Control.reveal(value.control),
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
        <Form.ControlChipField />
      </Base.Section>
      <Base.Section title="Actuation">
        <Base.Field<BaseButton.Mode> path="mode" label="Mode" optional>
          {({ value, onChange }) => (
            <BaseButton.SelectMode value={value} onChange={onChange} />
          )}
        </Base.Field>
        {/* The delay gates single-shot actuation (fire, pulse). Momentary's
            hold is the actuation, so the field is hidden there. */}
        {value.mode !== "momentary" && <Form.ActivationDelayField />}
      </Base.Section>
    </Base.Sections>
  );
};

export const ButtonForm = (): ReactElement => (
  <Form.Tabs tabs={["style", "control"]}>
    <Tabs.Content itemKey="style">
      <Base.Sections x>
        <Base.Section title="Label">
          <Label.Form
            path="label"
            omit={["align", "maxInlineSize", "level", "direction"]}
          />
        </Base.Section>
        <Base.Section title="Appearance">
          <Form.ColorField path="color" />
          <Form.SizeField />
        </Base.Section>
      </Base.Sections>
    </Tabs.Content>
    <Tabs.Content itemKey="control">
      <ButtonTelemForm path="" />
    </Tabs.Content>
  </Form.Tabs>
);
