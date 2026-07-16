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
import { Flex } from "@/flex";
import { Form as Base } from "@/form";
import { Input } from "@/input";
import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Tabs } from "@/tabs";

const SelectTelemForm = ({ path }: { path: string }): ReactElement => {
  const { value, onChange } =
    Base.useField<
      Pick<schematic.NodeConfigSelect, "commandChannel" | "control" | "disabled">
    >(path);

  const handleSinkChange = (v: channel.Key): void =>
    onChange({
      ...value,
      commandChannel: v,
      control: { ...value.control, show: true, showChip: true, showIndicator: true },
      disabled: v === 0,
    });

  return (
    <Form.Wrapper x grow align="stretch">
      <Input.Item label="Command channel" grow>
        <Channel.SelectSingle
          value={value.commandChannel ?? 0}
          onChange={handleSinkChange}
        />
      </Input.Item>
      <Form.ControlChipField />
    </Form.Wrapper>
  );
};

export const SelectForm = (): ReactElement => (
  <Tabs.Frame initialValue="style" grow>
    <Tabs.Selector>
      <Tabs.Tab itemKey="style">Style</Tabs.Tab>
      <Tabs.Tab itemKey="options">Options</Tabs.Tab>
      <Tabs.Tab itemKey="control">Control</Tabs.Tab>
    </Tabs.Selector>
    <Tabs.Content itemKey="style">
      <Form.Wrapper y align="stretch">
        <Flex.Box y align="stretch" grow gap="small">
          <Label.Form path="label" />
          <Flex.Box x>
            <Form.SizeField />
            <Form.ColorField path="color" />
            <Base.NumericField
              path="inlineSize"
              label="Width"
              inputProps={Form.VALUE_WIDTH_INPUT_PROPS}
            />
          </Flex.Box>
        </Flex.Box>
      </Form.Wrapper>
    </Tabs.Content>
    <Tabs.Content itemKey="options">
      <Form.Wrapper y align="stretch">
        <Form.StateMappingForm path="options" />
      </Form.Wrapper>
    </Tabs.Content>
    <Tabs.Content itemKey="control">
      <SelectTelemForm path="" />
    </Tabs.Content>
  </Tabs.Frame>
);
