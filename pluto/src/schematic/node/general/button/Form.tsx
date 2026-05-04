// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { type ReactElement, useCallback } from "react";

import { Channel } from "@/channel";
import { Flex } from "@/flex";
import { Form } from "@/form";
import { Input } from "@/input";
import { CommonStyleForm } from "@/schematic/node/common/forms";
import {
  ACTIVATION_DELAY_INPUT_PROPS,
  COMMON_TOGGLE_FORM_TABS,
  FormWrapper,
} from "@/schematic/node/common/forms/helpers";
import { type ControlStateProps } from "@/schematic/node/common/symbol/factories";
import { Tabs } from "@/tabs";
import { telem } from "@/telem/aether";
import { control } from "@/telem/control/aether";
import { Button as BaseButton } from "@/vis/button";

type ButtonTelemFormT = Omit<BaseButton.UseProps, "aetherKey"> & {
  control: ControlStateProps;
};

export const ButtonTelemForm = ({ path }: { path: string }): ReactElement => {
  const { value, onChange } = Form.useField<ButtonTelemFormT>(path);
  const sinkP = telem.sinkPipelinePropsZ.parse(value.sink?.props);
  const sink = control.setChannelValuePropsZ.parse(sinkP.segments.setter.props);

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
    <FormWrapper y empty>
      <Flex.Box x>
        <Input.Item label="Output Channel" grow padHelpText={false}>
          <Channel.SelectSingle value={sink.channel} onChange={handleSinkChange} />
        </Input.Item>
        <Form.NumericField
          label="Activation Delay"
          path="onClickDelay"
          inputProps={ACTIVATION_DELAY_INPUT_PROPS}
          hideIfNull
          padHelpText={false}
        />
        <Form.SwitchField
          path="control.show"
          label="Show Control Chip"
          hideIfNull
          optional
          padHelpText={false}
        />
      </Flex.Box>
      <Form.Field<BaseButton.Mode> path="mode" label="Mode" optional>
        {({ value, onChange }) => (
          <BaseButton.SelectMode value={value} onChange={onChange} />
        )}
      </Form.Field>
    </FormWrapper>
  );
};

export const ButtonForm = (): ReactElement => {
  const content: Tabs.RenderProp = useCallback(({ tabKey }) => {
    switch (tabKey) {
      case "control":
        return <ButtonTelemForm path="" />;
      default:
        return (
          <CommonStyleForm
            omit={["align", "maxInlineSize"]}
            hideInnerOrientation
            hideOuterOrientation
          />
        );
    }
  }, []);

  const props = Tabs.useStatic({ tabs: COMMON_TOGGLE_FORM_TABS, content });

  return <Tabs.Tabs {...props} />;
};
