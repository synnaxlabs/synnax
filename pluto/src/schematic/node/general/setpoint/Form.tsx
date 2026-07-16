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
import { Orientation } from "@/schematic/node/common/orientation";
import { Tabs } from "@/tabs";

export const SetpointTelemForm = ({ path }: { path: string }): ReactElement => {
  const { value, onChange } =
    Base.useField<
      Pick<
        schematic.NodeConfigSetpoint,
        "commandChannel" | "stateChannel" | "control" | "disabled"
      >
    >(path);
  const handleSinkChange = (v: channel.Key | null): void => {
    v ??= 0;
    onChange({
      ...value,
      commandChannel: v,
      stateChannel: v,
      control: { ...value.control, showChip: true, showIndicator: true },
      disabled: v == 0,
    });
  };

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

export const SetpointForm = (): ReactElement => (
  <Tabs.Frame initialValue="style">
    <Tabs.Selector>
      <Tabs.Tab itemKey="style">Style</Tabs.Tab>
      <Tabs.Tab itemKey="control">Control</Tabs.Tab>
    </Tabs.Selector>
    <Tabs.Content itemKey="style">
      <Form.Wrapper x align="stretch">
        <Flex.Box y align="stretch" grow gap="small">
          <Label.Form path="label" />
          <Flex.Box x>
            <Form.UnitsField />
            <Form.SizeField />
            <Form.ColorField path="color" />
          </Flex.Box>
        </Flex.Box>
        <Orientation.Field path="" hideInner />
      </Form.Wrapper>
    </Tabs.Content>
    <Tabs.Content itemKey="control">
      <SetpointTelemForm path="" />
    </Tabs.Content>
  </Tabs.Frame>
);
