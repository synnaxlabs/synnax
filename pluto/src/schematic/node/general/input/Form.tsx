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
  const sinkP = zod.parse(telem.sinkPipelinePropsZ, value.sink?.props, {
    label: "sink pipeline",
  });
  const sink = zod.parse(control.setChannelValuePropsZ, sinkP.segments.setter.props, {
    label: "setter sink",
  });

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
    <Base.Sections x>
      <Base.Section title="Command">
        <Input.Item label="Channel" padHelpText={false}>
          <Channel.SelectSingle value={sink.channel} onChange={handleSinkChange} />
        </Input.Item>
        <Form.ControlChipField />
      </Base.Section>
    </Base.Sections>
  );
};

export const InputForm = (): ReactElement => (
  <Form.Tabs tabs={["style", "control"]}>
    <Tabs.Content itemKey="style">
      <Base.Sections x>
        <Base.Section title="Label">
          <Label.Form path="label" />
        </Base.Section>
        <Base.Section title="Appearance">
          <Form.ColorField path="color" />
          <Form.SizeField />
        </Base.Section>
      </Base.Sections>
    </Tabs.Content>
    <Tabs.Content itemKey="control">
      <InputTelemForm path="" />
    </Tabs.Content>
  </Form.Tabs>
);
