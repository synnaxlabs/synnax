// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Input } from "@synnaxlabs/charon/input";
import { Tabs } from "@synnaxlabs/charon/tabs";
import { type channel } from "@synnaxlabs/client";
import { type ReactElement, useCallback } from "react";

import { Channel } from "@/channel";
import { Form as Base } from "@synnaxlabs/charon/form";
import { Form } from "@/schematic/node/common/form";
import { telem } from "@/telem/aether";
import { type Toggle } from "@/vis/toggle";
interface LightTelemFormT extends Omit<Toggle.UseProps, "aetherKey"> {}

const LightTelemForm = ({ path }: { path: string }): ReactElement => {
  const { value, onChange } = Base.useField<LightTelemFormT>(path);
  const sourceP = telem.sourcePipelinePropsZ.parse(value.source?.props);
  const source = telem.streamChannelValuePropsZ.parse(
    sourceP.segments.valueStream.props,
  );
  const threshold = telem.withinBoundsProps.parse(sourceP.segments.threshold.props);

  const handleSourceChange = (v: channel.Key | null): void => {
    v ??= 0;
    const t = telem.sourcePipeline("boolean", {
      connections: [{ from: "valueStream", to: "threshold" }],
      segments: {
        valueStream: telem.streamChannelValue({ channel: v }),
        threshold: telem.withinBounds({ trueBound: threshold.trueBound }),
      },
      outlet: "threshold",
    });
    onChange({ ...value, source: t });
  };

  const handleThresholdChange = (bounds: { lower: number; upper: number }): void => {
    const t = telem.sourcePipeline("boolean", {
      connections: [{ from: "valueStream", to: "threshold" }],
      segments: {
        valueStream: telem.streamChannelValue({ channel: source.channel }),
        threshold: telem.withinBounds({ trueBound: bounds }),
      },
      outlet: "threshold",
    });
    onChange({ ...value, source: t });
  };
  if (typeof source.channel !== "number")
    throw new Error("Channel key must be used for light telemetry");

  return (
    <Form.Wrapper x align="stretch">
      <Input.Item label="Input Channel" grow>
        <Channel.SelectSingle value={source.channel} onChange={handleSourceChange} />
      </Input.Item>
      <Input.Item label="Lower Threshold">
        <Input.Numeric
          value={threshold.trueBound.lower ?? 0.9}
          onChange={(v) => handleThresholdChange({ ...threshold.trueBound, lower: v })}
        />
      </Input.Item>
      <Input.Item label="Upper Threshold">
        <Input.Numeric
          value={threshold.trueBound.upper ?? 1.1}
          onChange={(v) => handleThresholdChange({ ...threshold.trueBound, upper: v })}
        />
      </Input.Item>
    </Form.Wrapper>
  );
};

const LIGHT_FORM_TABS: Tabs.Tab[] = [
  { tabKey: "style", name: "Style" },
  { tabKey: "telemetry", name: "Telemetry" },
];

export const LightForm = (): ReactElement => {
  const content: Tabs.RenderProp = useCallback(({ tabKey }) => {
    switch (tabKey) {
      case "telemetry":
        return <LightTelemForm path="" />;
      default:
        return <Form.StyleForm />;
    }
  }, []);
  const props = Tabs.useStatic({ tabs: LIGHT_FORM_TABS, content });
  return <Tabs.Tabs {...props} />;
};
