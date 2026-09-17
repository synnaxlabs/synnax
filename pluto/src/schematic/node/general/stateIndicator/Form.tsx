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
import { Label } from "@/schematic/node/common/label";
import { Tabs } from "@/tabs";
import { telem } from "@/telem/aether";
import { Staleness } from "@/vis/staleness";
import { type StateIndicator as BaseStateIndicator } from "@/vis/stateIndicator";
const StateIndicatorTelemForm = ({ path }: { path: string }): ReactElement => {
  const { value, onChange } =
    Base.useField<Omit<BaseStateIndicator.UseProps, "aetherKey">>(path);
  const sourceP = zod.parse(telem.sourcePipelinePropsZ, value.source?.props, {
    label: "source pipeline",
  });
  const source = zod.parse(
    telem.streamChannelValuePropsZ,
    sourceP.segments.valueStream.props,
    { label: "value stream source" },
  );

  const handleSourceChange = (v: channel.Key | null): void => {
    v ??= 0;
    const t = telem.sourcePipeline("number", {
      connections: [],
      segments: { valueStream: telem.streamChannelValue({ channel: v }) },
      outlet: "valueStream",
    });
    onChange({ ...value, source: t });
  };

  return (
    <Base.Sections x>
      <Base.Section title="State">
        <Input.Item label="Channel" padHelpText={false}>
          <Channel.SelectSingle
            value={source.channel as number}
            onChange={handleSourceChange}
          />
        </Input.Item>
      </Base.Section>
      <Base.Section title="Staleness">
        <Staleness.Fields />
      </Base.Section>
    </Base.Sections>
  );
};

export const StateIndicatorForm = (): ReactElement => (
  <Form.Tabs tabs={["style", "telemetry", "options"]}>
    <Tabs.Content itemKey="style">
      <Base.Sections x>
        <Base.Section title="Label">
          <Label.Form path="label" />
        </Base.Section>
        <Base.Section title="Appearance">
          <Form.ColorField path="color" />
          <Base.NumericField
            path="inlineSize"
            label="Width"
            hideIfNull
            padHelpText={false}
            inputProps={Form.VALUE_WIDTH_INPUT_PROPS}
          />
        </Base.Section>
      </Base.Sections>
    </Tabs.Content>
    <Tabs.Content itemKey="telemetry">
      <StateIndicatorTelemForm path="" />
    </Tabs.Content>
    <Tabs.Content itemKey="options">
      <Form.StateMappingForm path="options" showColor />
    </Tabs.Content>
  </Form.Tabs>
);
