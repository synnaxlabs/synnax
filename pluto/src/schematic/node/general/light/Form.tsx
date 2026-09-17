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
import { Form } from "@/schematic/node/common/form";
import { Tabs } from "@/tabs";
import { telem } from "@/telem/aether";
import { Staleness } from "@/vis/staleness";
import { type Toggle } from "@/vis/toggle";
interface LightTelemFormT extends Omit<Toggle.UseProps, "aetherKey"> {}

const LightTelemForm = ({ path }: { path: string }): ReactElement => {
  const { value, onChange } = Base.useField<LightTelemFormT>(path);
  const sourceP = zod.parse(telem.sourcePipelinePropsZ, value.source?.props, {
    label: "source pipeline",
  });
  const source = zod.parse(
    telem.streamChannelValuePropsZ,
    sourceP.segments.valueStream.props,
    { label: "value stream source" },
  );
  const threshold = zod.parse(
    telem.withinBoundsProps,
    sourceP.segments.threshold.props,
    { label: "threshold source" },
  );

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
    <Base.Sections x>
      <Base.Section title="State">
        <Input.Item label="Channel" padHelpText={false}>
          <Channel.SelectSingle value={source.channel} onChange={handleSourceChange} />
        </Input.Item>
        <Input.Item label="Lower threshold" padHelpText={false}>
          <Input.Numeric
            value={threshold.trueBound.lower ?? 0.9}
            onChange={(v) =>
              handleThresholdChange({ ...threshold.trueBound, lower: v })
            }
          />
        </Input.Item>
        <Input.Item label="Upper threshold" padHelpText={false}>
          <Input.Numeric
            value={threshold.trueBound.upper ?? 1.1}
            onChange={(v) =>
              handleThresholdChange({ ...threshold.trueBound, upper: v })
            }
          />
        </Input.Item>
      </Base.Section>
      <Base.Section title="Staleness">
        <Staleness.Fields />
      </Base.Section>
    </Base.Sections>
  );
};

export const LightForm = (): ReactElement => (
  <Form.Tabs tabs={["style", "telemetry"]}>
    <Tabs.Content itemKey="style">
      <Form.StyleForm />
    </Tabs.Content>
    <Tabs.Content itemKey="telemetry">
      <LightTelemForm path="" />
    </Tabs.Content>
  </Form.Tabs>
);
