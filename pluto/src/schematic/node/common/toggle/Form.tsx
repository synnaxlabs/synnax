// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as Base } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { type ReactElement } from "react";

import { Channel } from "@/channel";
import { type Control } from "@/schematic/node/common/control";
import { ACTIVATION_DELAY_INPUT_PROPS } from "@/schematic/node/common/form/input";
import { Wrapper } from "@/schematic/node/common/form/Wrapper";
import { telem } from "@/telem/aether";
import { control } from "@/telem/control/aether";
import { type Toggle as VisToggle } from "@/vis/toggle";

interface ChannelFormProps {
  path: string;
  omit?: string[];
}

export const ChannelForm = ({ path, omit = [] }: ChannelFormProps): ReactElement => {
  const { value, onChange } = Base.useField<
    Omit<VisToggle.UseProps, "aetherKey"> & { control: Control.StateProps }
  >(path);
  const sourceP = telem.sourcePipelinePropsZ.parse(value.source?.props);
  const sinkP = telem.sinkPipelinePropsZ.parse(value.sink?.props);
  const source = telem.streamChannelValuePropsZ.parse(
    sourceP.segments.valueStream.props,
  );
  const sink = control.setChannelValuePropsZ.parse(sinkP.segments.setter.props);

  const handleSourceChange = (v: channel.Key | null): void => {
    v ??= 0;
    const t = telem.sourcePipeline("boolean", {
      connections: [{ from: "valueStream", to: "threshold" }],
      segments: {
        valueStream: telem.streamChannelValue({ channel: v }),
        threshold: telem.withinBounds({ trueBound: { lower: 0.9, upper: 1.1 } }),
      },
      outlet: "threshold",
    });
    onChange({ ...value, source: t });
  };

  const handleSinkChange = (v: channel.Key | null): void => {
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
        showChip: true,
        showIndicator: true,
        ...value.control,
        chip: { sink: controlChipSink, source: authSource },
        indicator: { statusSource: authSource },
      },
    });
  };

  return (
    <Wrapper y empty>
      <Flex.Box x grow>
        <Input.Item label="State Channel" grow padHelpText={false}>
          <Channel.SelectSingle
            value={source.channel as number}
            onChange={handleSourceChange}
          />
        </Input.Item>
        <Input.Item label="Command Channel" grow padHelpText={false}>
          <Channel.SelectSingle value={sink.channel} onChange={handleSinkChange} />
        </Input.Item>
      </Flex.Box>
      <Flex.Box x grow>
        {!omit.includes("onClickDelay") && (
          <Base.NumericField
            label="Activation Delay"
            path="onClickDelay"
            grow
            inputProps={ACTIVATION_DELAY_INPUT_PROPS}
            hideIfNull
            padHelpText={false}
          />
        )}
        <Base.SwitchField
          path="control.show"
          label="Show Control Chip"
          hideIfNull
          optional
          padHelpText={false}
        />
      </Flex.Box>
    </Wrapper>
  );
};
