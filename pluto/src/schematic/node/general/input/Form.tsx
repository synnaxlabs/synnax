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
import { Form as Base } from "@/form";
import { Input } from "@/input";
import { type Control } from "@/schematic/node/common/control";
import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Tabs } from "@/tabs";
import { telem } from "@/telem/aether";
import { control } from "@/telem/control/aether";
import { type Input as BaseInput } from "@/vis/input";
interface InputTelemFormProps {
  path: string;
}

const InputTelemForm = ({ path }: InputTelemFormProps): ReactElement => {
  const { value, onChange } = Base.useField<
    Omit<BaseInput.UseProps, "aetherKey"> & {
      control: Control.StateProps;
      disabled?: boolean;
    }
  >(path);
  const sinkP = telem.sinkPipelinePropsZ.parse(value.sink?.props);
  const sink = control.setChannelValuePropsZ.parse(sinkP.segments.setter.props);

  const handleSinkChange = (v: channel.Key | null): void => {
    v ??= 0;
    const t = telem.sinkPipeline("string", {
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
        showChip: true,
        chip: { sink: controlChipSink, source: authSource },
        showIndicator: true,
        indicator: { statusSource: authSource },
      },
      disabled: v == 0,
    });
  };

  return (
    <Form.Wrapper x grow align="stretch">
      <Input.Item label="Command channel" grow>
        <Channel.SelectSingle value={sink.channel} onChange={handleSinkChange} />
      </Input.Item>
      <Base.SwitchField
        path="control.show"
        label="Show control chip"
        hideIfNull
        optional
        padHelpText={false}
      />
    </Form.Wrapper>
  );
};

export const InputForm = (): ReactElement => {
  const content: Tabs.RenderProp = useCallback(({ tabKey }) => {
    switch (tabKey) {
      case "control":
        return <InputTelemForm path="" />;
      default:
        return (
          <Form.Wrapper x>
            <Flex.Box y align="stretch" grow gap="small">
              <Label.Form path="label" />
              <Flex.Box x>
                <Base.Field<Component.Size>
                  path="size"
                  label="Size"
                  hideIfNull
                  padHelpText={false}
                >
                  {({ value, onChange }) => (
                    <Component.SelectSize value={value} onChange={onChange} />
                  )}
                </Base.Field>
                <Form.ColorField path="color" />
              </Flex.Box>
            </Flex.Box>
          </Form.Wrapper>
        );
    }
  }, []);
  const props = Tabs.useStatic({ tabs: Form.COMMON_TOGGLE_FORM_TABS, content });
  return <Tabs.Tabs {...props} />;
};
