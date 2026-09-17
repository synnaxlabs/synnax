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
    <Base.Sections x>
      <Base.Section title="Command">
        <Input.Item label="Channel" padHelpText={false}>
          <Channel.SelectSingle value={sink.channel} onChange={handleSinkChange} />
        </Input.Item>
        <Form.ControlChipField />
      </Base.Section>
      <Base.Section title="Actuation">
        <Base.Field<BaseButton.Mode> path="mode" label="Mode" optional>
          {({ value, onChange }) => (
            <BaseButton.SelectMode value={value} onChange={onChange} />
          )}
        </Base.Field>
        {/* The delay gates single-shot actuation (fire, pulse). Momentary's
            hold is the actuation, so the field is hidden there. */}
        {mode !== "momentary" && <Form.ActivationDelayField />}
      </Base.Section>
    </Base.Sections>
  );
};

export const ButtonForm = (): ReactElement => (
  <Form.Tabs tabs={["style", "control"]}>
    <Tabs.Content itemKey="style">
      <Base.Sections x>
        <Base.Section title="Label">
          <Label.Form
            path="label"
            omit={["align", "maxInlineSize", "level", "direction"]}
          />
        </Base.Section>
        <Base.Section title="Appearance">
          <Form.ColorField path="color" />
          <Form.SizeField defaultValue="medium" />
        </Base.Section>
      </Base.Sections>
    </Tabs.Content>
    <Tabs.Content itemKey="control">
      <ButtonTelemForm path="" />
    </Tabs.Content>
  </Form.Tabs>
);
