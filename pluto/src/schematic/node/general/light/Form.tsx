// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type schematic } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { Channel } from "@/channel";
import { Form as Base } from "@/form";
import { Input } from "@/input";
import { Form } from "@/schematic/node/common/form";
import { Telem } from "@/schematic/node/common/telem";
import { Tabs } from "@/tabs";
import { Staleness } from "@/vis/staleness";

type LightTelemFormT = Pick<schematic.LightNodeConfig, "channel" | "threshold">;

const LightTelemForm = ({ path }: { path: string }): ReactElement => {
  const { value, onChange } = Base.useField<LightTelemFormT>(path);
  const threshold = value.threshold ?? Telem.DEFAULT_THRESHOLD;

  const handleSourceChange = (v: channel.Key | null): void =>
    onChange({ ...value, channel: v ?? undefined });

  const handleThresholdChange = (bounds: { lower: number; upper: number }): void =>
    onChange({ ...value, threshold: bounds });

  return (
    <Base.Sections x>
      <Base.Section title="State">
        <Input.Item label="Channel" padHelpText={false}>
          <Channel.SelectSingle
            value={value.channel ?? 0}
            onChange={handleSourceChange}
          />
        </Input.Item>
        <Input.Item label="Lower threshold" padHelpText={false}>
          <Input.Numeric
            value={threshold.lower}
            onChange={(v) => handleThresholdChange({ ...threshold, lower: v })}
          />
        </Input.Item>
        <Input.Item label="Upper threshold" padHelpText={false}>
          <Input.Numeric
            value={threshold.upper}
            onChange={(v) => handleThresholdChange({ ...threshold, upper: v })}
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
