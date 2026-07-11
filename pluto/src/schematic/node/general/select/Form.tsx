// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { Channel } from "@/channel";
import { Flex } from "@/flex";
import { Form as Base } from "@/form";
import { Input } from "@/input";
import { type Control } from "@/schematic/node/common/control";
import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Tabs } from "@/tabs";
import { telem } from "@/telem/aether";
import { control } from "@/telem/control/aether";
import { type Setpoint } from "@/vis/setpoint";

const SelectTelemForm = ({ path }: { path: string }): ReactElement => {
  const { value, onChange } = Base.useField<
    Pick<Setpoint.UseProps, "sink"> & {
      control: Control.StateProps;
      disabled?: boolean;
    }
  >(path);
  const sinkP = telem.sinkPipelinePropsZ.parse(value.sink?.props);
  const sink = control.setChannelValuePropsZ.parse(sinkP.segments.setter.props);

  const handleSinkChange = (v: channel.Key): void => {
    const t = telem.sinkPipeline("number", {
      connections: [],
      segments: { setter: control.setChannelValue({ channel: v }) },
      inlet: "setter",
    });

    const authSource = control.authoritySource({ channel: v });

    const controlChipSink = control.acquireChannelControl({
      channel: v,
      authority: 255,
    });

    onChange({
      ...value,
      sink: t,
      control: {
        ...value.control,
        show: true,
        showChip: true,
        chip: { sink: controlChipSink, source: authSource },
        showIndicator: true,
        indicator: { statusSource: authSource },
      },
      disabled: v === 0,
    });
  };

  return (
    <Form.Wrapper x grow align="stretch">
      <Input.Item label="Command channel" grow>
        <Channel.SelectSingle value={sink.channel} onChange={handleSinkChange} />
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
