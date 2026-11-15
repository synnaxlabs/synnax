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
import { type ColorFallbacks, type FormProps } from "@/schematic/node/spec";
import { Button as BaseButton } from "@/vis/button";
import { Properties } from "@/vis/properties";

const path = deep.path<schematic.ButtonNodeConfig>();

export const colorFallbacks = {
  fillColor: Form.primaryFallback,
} satisfies ColorFallbacks;

const ButtonTelemForm = (): ReactElement => {
  const { value, onChange } = Base.useField(path);
  const handleSinkChange = (v: channel.Key | null): void =>
    onChange({
      ...value,
      commandChannel: v ?? undefined,
    });

  return (
    <Base.Sections x>
      <Base.Section title="Command">
        <Input.Item label="Channel" padHelpText={false}>
          <Channel.SelectSingle
            allowNone
            value={value.commandChannel ?? 0}
            onChange={handleSinkChange}
          />
        </Input.Item>
        <Form.ControlChipField />
      </Base.Section>
      <Base.Section title="Actuation">
        <Base.Field path={path.mode} label="Mode">
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

export const ButtonForm = ({ tab, onTabChange }: FormProps): ReactElement => (
  <Properties.Tabs tabs={["control", "style"]} tab={tab} onTabChange={onTabChange}>
    <Tabs.Content itemKey="style">
      <Base.Sections x>
        <Base.Section title="Label">
          <Label.Form path={path.label} omit={["level", "direction"]} multiline />
        </Base.Section>
        <Base.Section title="Appearance">
          <Form.FillField fallback={colorFallbacks.fillColor} />
          <Form.ScaledSizeField />
        </Base.Section>
      </Base.Sections>
    </Tabs.Content>
    <Tabs.Content itemKey="control">
      <ButtonTelemForm />
    </Tabs.Content>
  </Properties.Tabs>
);
