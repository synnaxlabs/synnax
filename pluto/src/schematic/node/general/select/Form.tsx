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
import { Component } from "@/component";
import { Flex } from "@/flex";
import { Form } from "@/form";
import { Input } from "@/input";
import {
  ColorControl,
  FormWrapper,
  LabelControls,
  StateMappingForm,
  valueWidthInputProps,
} from "@/schematic/node/common/forms";
import { type ControlStateProps } from "@/schematic/node/common/symbol/factories";
import { Tabs } from "@/tabs";
import { telem } from "@/telem/aether";
import { control } from "@/telem/control/aether";
import { type Setpoint } from "@/vis/setpoint";

const SelectTelemForm = ({ path }: { path: string }): ReactElement => {
  const { value, onChange } = Form.useField<
    Pick<Setpoint.UseProps, "sink"> & {
      control: ControlStateProps;
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
    <FormWrapper x grow align="stretch">
      <Input.Item label="Command Channel" grow>
        <Channel.SelectSingle value={sink.channel} onChange={handleSinkChange} />
      </Input.Item>
      <Form.SwitchField
        path="control.show"
        label="Show Control Chip"
        hideIfNull
        optional
      />
    </FormWrapper>
  );
};

const SELECT_FORM_TABS: Tabs.Tab[] = [
  { tabKey: "style", name: "Style" },
  { tabKey: "options", name: "Options" },
  { tabKey: "control", name: "Control" },
];

export const SelectForm = (): ReactElement => {
  const content: Tabs.RenderProp = useCallback(({ tabKey }) => {
    switch (tabKey) {
      case "control":
        return <SelectTelemForm path="" />;
      case "options":
        return (
          <FormWrapper y align="stretch">
            <StateMappingForm path="options" />
          </FormWrapper>
        );
      default:
        return (
          <FormWrapper y align="stretch">
            <Flex.Box y align="stretch" grow gap="small">
              <LabelControls path="label" />
              <Flex.Box x>
                <Form.Field<Component.Size>
                  path="size"
                  label="Size"
                  hideIfNull
                  padHelpText={false}
                >
                  {({ value, onChange }) => (
                    <Component.SelectSize value={value} onChange={onChange} />
                  )}
                </Form.Field>
                <ColorControl path="color" />
                <Form.NumericField
                  path="inlineSize"
                  label="Width"
                  inputProps={valueWidthInputProps}
                />
              </Flex.Box>
            </Flex.Box>
          </FormWrapper>
        );
    }
  }, []);
  const props = Tabs.useStatic({ tabs: SELECT_FORM_TABS, content });
  return <Tabs.Tabs {...props} grow />;
};
