// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Component } from "@synnaxlabs/charon/component";
import { Flex } from "@synnaxlabs/charon/flex";
import { Input } from "@synnaxlabs/charon/input";
import { Select } from "@synnaxlabs/charon/select";
import { Tabs } from "@synnaxlabs/charon/tabs";
import { type channel } from "@synnaxlabs/client";
import { type ReactElement, useCallback } from "react";

import { Channel } from "@/channel";
import { Form as Base } from "@synnaxlabs/charon/form";
import { type Control } from "@/schematic/node/common/control";
import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Orientation } from "@/schematic/node/common/orientation";
import { telem } from "@/telem/aether";
import { control } from "@/telem/control/aether";
import { type Setpoint } from "@/vis/setpoint";

export const SetpointTelemForm = ({ path }: { path: string }): ReactElement => {
  const { value, onChange } = Base.useField<
    Omit<Setpoint.UseProps, "aetherKey"> & {
      control: Control.StateProps;
      disabled?: boolean;
    }
  >(path);
  const sinkP = telem.sinkPipelinePropsZ.parse(value.sink?.props);
  const sink = control.setChannelValuePropsZ.parse(sinkP.segments.setter.props);

  const handleSinkChange = (v: channel.Key | null): void => {
    v ??= 0;
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
      <Input.Item label="Command Channel" grow>
        <Channel.SelectSingle value={sink.channel} onChange={handleSinkChange} />
      </Input.Item>
    </Form.Wrapper>
  );
};

export const SetpointForm = (): ReactElement => {
  const content: Tabs.RenderProp = useCallback(({ tabKey }) => {
    switch (tabKey) {
      case "control":
        return <SetpointTelemForm path="" />;
      default:
        return (
          <Form.Wrapper x align="stretch">
            <Flex.Box y align="stretch" grow gap="small">
              <Label.Form path="label" />
              <Flex.Box x>
                <Base.TextField
                  path="units"
                  label="Units"
                  align="start"
                  padHelpText={false}
                />
                <Base.Field<Component.Size>
                  path="size"
                  label="Size"
                  hideIfNull
                  padHelpText={false}
                >
                  {({ value, onChange }) => (
                    <Select.SelectSize value={value} onChange={onChange} />
                  )}
                </Base.Field>
                <Form.ColorField path="color" />
              </Flex.Box>
            </Flex.Box>
            <Orientation.Field path="" hideInner />
          </Form.Wrapper>
        );
    }
  }, []);

  const props = Tabs.useStatic({ tabs: Form.COMMON_TOGGLE_FORM_TABS, content });

  return <Tabs.Tabs {...props} />;
};
