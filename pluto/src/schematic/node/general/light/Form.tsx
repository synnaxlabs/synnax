// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type schematic } from "@synnaxlabs/client";
import { Form as Base } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Tabs } from "@synnaxlabs/lyra/tabs";
import { type ReactElement } from "react";

import { Channel } from "@/channel";
import { Form } from "@/schematic/node/common/form";
import { Label } from "@/schematic/node/common/label";
import { Orientation } from "@/schematic/node/common/orientation";
import { Telem } from "@/schematic/node/common/telem";
import { type FormProps } from "@/schematic/node/spec";
import { Properties } from "@/vis/properties";
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

const StyleForm = (): ReactElement => {
  // An absent on color paints the stroke color.
  const strokeColor = Base.useFieldValue<schematic.LightNodeConfig["strokeColor"]>(
    "strokeColor",
    { optional: true },
  );
  return (
    <Base.Sections x>
      <Base.Section title="Label">
        <Label.Form path="label" />
      </Base.Section>
      <Base.Section title="Appearance">
        <Form.ColorField path="strokeColor" label="Stroke" />
        <Form.ColorField
          path="onColor"
          label="On"
          fallback={strokeColor ?? undefined}
        />
        <Form.ScaleField path="scale" />
      </Base.Section>
      <Orientation.Section path="" />
    </Base.Sections>
  );
};

export const LightForm = ({ tab, onTabChange }: FormProps): ReactElement => (
  <Properties.Tabs tabs={["telemetry", "style"]} tab={tab} onTabChange={onTabChange}>
    <Tabs.Content itemKey="style">
      <StyleForm />
    </Tabs.Content>
    <Tabs.Content itemKey="telemetry">
      <LightTelemForm path="" />
    </Tabs.Content>
  </Properties.Tabs>
);
