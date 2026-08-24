// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { zod } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Channel } from "@/channel";
import { Flex } from "@/flex";
import { Form as Base } from "@/form";
import { Input } from "@/input";
import { type Control } from "@/schematic/node/common/control";
import { Form } from "@/schematic/node/common/form";
import { Tabs } from "@/tabs";
import { telem } from "@/telem/aether";
import { control } from "@/telem/control/aether";
import { Button as BaseButton } from "@/vis/button";

type ButtonTelemFormT = Omit<BaseButton.UseProps, "aetherKey"> & {
  control: Control.StateProps;
};

export const ButtonTelemForm = ({ path }: { path: string }): ReactElement => {
  const { value, onChange } = Base.useField<ButtonTelemFormT>(path);
  const mode = Base.useFieldValue<BaseButton.Mode>("mode", { optional: true });
  const sinkP = zod.parse(telem.sinkPipelinePropsZ, value.sink?.props, {
    label: "sink pipeline",
  });
  const sink = zod.parse(control.setChannelValuePropsZ, sinkP.segments.setter.props, {
    label: "setter sink",
  });

  const handleSinkChange = (v: channel.Key): void => {
    v ??= 0;
    const t = telem.sinkPipeline("boolean", {
      connections: [{ from: "setpoint", to: "setter" }],
      segments: {
        setter: control.setChannelValue({ channel: v }),
        setpoint: telem.setpoint({ truthy: 1, falsy: 0 }),
      },
      inlet: "setpoint",
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
        showChip: true,
        chip: { sink: controlChipSink, source: authSource },
        showIndicator: true,
        indicator: { statusSource: authSource },
      },
    });
  };

  return (
    <Form.Wrapper y empty>
      <Flex.Box x>
        <Input.Item label="Channel" grow padHelpText={false}>
          <Channel.SelectSingle value={sink.channel} onChange={handleSinkChange} />
        </Input.Item>
        {/* The delay gates single-shot actuation (fire, pulse). Momentary's
            hold is the actuation, so the field is hidden there. */}
        {mode !== "momentary" && <Form.ActivationDelayField />}
        <Form.ControlChipField />
      </Flex.Box>
      <Base.Field<BaseButton.Mode> path="mode" label="Mode" optional>
        {({ value, onChange }) => (
          <BaseButton.SelectMode value={value} onChange={onChange} />
        )}
      </Base.Field>
    </Form.Wrapper>
  );
};

export const ButtonForm = (): ReactElement => (
  <Tabs.Frame initialValue="style">
    <Tabs.Selector>
      <Tabs.Tab itemKey="style">Style</Tabs.Tab>
      <Tabs.Tab itemKey="control">Control</Tabs.Tab>
    </Tabs.Selector>
    <Tabs.Content itemKey="style">
      <Form.StyleForm
        omit={["align", "maxInlineSize"]}
        hideInnerOrientation
        hideOuterOrientation
      />
    </Tabs.Content>
    <Tabs.Content itemKey="control">
      <ButtonTelemForm path="" />
    </Tabs.Content>
  </Tabs.Frame>
);
