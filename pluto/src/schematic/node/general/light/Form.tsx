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
import * as CommonTelem from "@/schematic/node/common/telem";
import { Tabs } from "@/tabs";
type LightTelemFormT = Pick<schematic.NodeConfigLight, "channel" | "threshold">;

const LightTelemForm = ({ path }: { path: string }): ReactElement => {
  const { value, onChange } = Base.useField<LightTelemFormT>(path);
  const threshold = value.threshold ?? CommonTelem.DEFAULT_THRESHOLD;

  const handleSourceChange = (v: channel.Key | null): void =>
    onChange({ ...value, channel: v ?? undefined });

  const handleThresholdChange = (bounds: { lower: number; upper: number }): void =>
    onChange({ ...value, threshold: bounds });

  return (
    <Form.Wrapper x align="stretch">
      <Input.Item label="Input channel" grow>
        <Channel.SelectSingle
          value={value.channel ?? 0}
          onChange={handleSourceChange}
        />
      </Input.Item>
      <Input.Item label="Lower threshold">
        <Input.Numeric
          value={threshold.lower ?? 0.9}
          onChange={(v) => handleThresholdChange({ ...threshold, lower: v })}
        />
      </Input.Item>
      <Input.Item label="Upper threshold">
        <Input.Numeric
          value={threshold.upper ?? 1.1}
          onChange={(v) => handleThresholdChange({ ...threshold, upper: v })}
        />
      </Input.Item>
    </Form.Wrapper>
  );
};

export const LightForm = (): ReactElement => (
  <Tabs.Frame initialValue="style">
    <Tabs.Selector>
      <Tabs.Tab itemKey="style">Style</Tabs.Tab>
      <Tabs.Tab itemKey="telemetry">Telemetry</Tabs.Tab>
    </Tabs.Selector>
    <Tabs.Content itemKey="style">
      <Form.StyleForm />
    </Tabs.Content>
    <Tabs.Content itemKey="telemetry">
      <LightTelemForm path="" />
    </Tabs.Content>
  </Tabs.Frame>
);
