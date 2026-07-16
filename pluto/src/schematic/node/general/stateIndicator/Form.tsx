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
const StateIndicatorTelemForm = ({ path }: { path: string }): ReactElement => {
  const { value, onChange } =
    Base.useField<Pick<schematic.NodeConfigStateIndicator, "channel">>(path);

  const handleSourceChange = (v: channel.Key | null): void =>
    onChange({ ...value, channel: v ?? undefined });

  return (
    <Form.Wrapper x grow align="stretch">
      <Input.Item label="Input channel" grow>
        <Channel.SelectSingle
          value={value.channel ?? 0}
          onChange={handleSourceChange}
        />
      </Input.Item>
    </Form.Wrapper>
  );
};

export const StateIndicatorForm = (): ReactElement => (
  <Tabs.Frame initialValue="style" grow>
    <Tabs.Selector>
      <Tabs.Tab itemKey="style">Style</Tabs.Tab>
      <Tabs.Tab itemKey="options">Options</Tabs.Tab>
      <Tabs.Tab itemKey="telemetry">Telemetry</Tabs.Tab>
    </Tabs.Selector>
    <Tabs.Content itemKey="style">
      <Form.Wrapper y align="stretch">
        <Flex.Box y align="stretch" grow gap="small">
          <Label.Form path="label" />
          <Flex.Box x>
            <Form.ColorField path="color" />
            <Base.NumericField
              path="inlineSize"
              label="Width"
              hideIfNull
              inputProps={Form.VALUE_WIDTH_INPUT_PROPS}
            />
          </Flex.Box>
        </Flex.Box>
      </Form.Wrapper>
    </Tabs.Content>
    <Tabs.Content itemKey="options">
      <Form.Wrapper y align="stretch">
        <Form.StateMappingForm path="options" showColor />
      </Form.Wrapper>
    </Tabs.Content>
    <Tabs.Content itemKey="telemetry">
      <StateIndicatorTelemForm path="" />
    </Tabs.Content>
  </Tabs.Frame>
);
