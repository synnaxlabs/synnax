// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { Color } from "@synnaxlabs/lyra/color";
import { type Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Tabs } from "@synnaxlabs/lyra/tabs";
import { Theming } from "@synnaxlabs/lyra/theming";
import { color, type notation, type text } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Channel } from "@/channel";
import { Notation } from "@/notation";
import { Properties } from "@/vis/properties";
import { Staleness } from "@/vis/staleness";
import { Value } from "@/vis/value";

/** Props for a cell variant's form. */
export interface FormProps extends Properties.SelectionProps {}

interface TelemFormT {
  channel: channel.Key;
  rollingAverage: number;
  precision?: number;
  notation: notation.Notation;
}

const TelemForm = () => {
  const { value, onChange } = Form.useField<TelemFormT>("");
  return (
    <>
      <Form.Section title="Source">
        <Input.Item label="Channel" padHelpText={false}>
          <Channel.SelectSingle
            value={value.channel}
            onChange={(key: channel.Key | null) =>
              onChange({ ...value, channel: key ?? 0 })
            }
          />
        </Input.Item>
        <Input.Item label="Averaging window" padHelpText={false}>
          <Input.Numeric
            value={value.rollingAverage}
            bounds={{ lower: 1, upper: 100 }}
            onChange={(rollingAverage) => onChange({ ...value, rollingAverage })}
          />
        </Input.Item>
      </Form.Section>
      <Form.Section title="Format">
        <Input.Item label="Notation" padHelpText={false}>
          <Notation.Select
            value={value.notation}
            onChange={(next: notation.Notation) =>
              onChange({ ...value, notation: next })
            }
          />
        </Input.Item>
        <Input.Item label="Precision" padHelpText={false}>
          <Input.Numeric
            value={value.precision ?? 2}
            bounds={{ lower: 0, upper: 10 }}
            onChange={(precision) => onChange({ ...value, precision })}
          />
        </Input.Item>
      </Form.Section>
      <Form.Section title="Staleness">
        <Staleness.Fields />
      </Form.Section>
    </>
  );
};

export const ValueForm = ({ tab, onTabChange }: FormProps) => {
  const theme = Theming.use();
  return (
    <Properties.Tabs
      tabs={["telemetry", "style", "redline"]}
      tab={tab}
      onTabChange={onTabChange}
    >
      <Tabs.Content itemKey="style">
        <Form.Sections x>
          <Form.Section title="Appearance">
            <Color.Field
              path="textColor"
              label="Text"
              fallback={theme.colors.gray.l11}
            />
            <Color.Field path="fillColor" label="Fill" fallback={color.ZERO} />
            <Form.Field<text.Level>
              path="level"
              label="Size"
              hideIfNull
              padHelpText={false}
            >
              {(p) => <Select.Text.Level {...p} />}
            </Form.Field>
          </Form.Section>
        </Form.Sections>
      </Tabs.Content>
      <Tabs.Content itemKey="telemetry">
        <Form.Sections x>
          <TelemForm />
        </Form.Sections>
      </Tabs.Content>
      <Tabs.Content itemKey="redline">
        <Value.RedlineForm path="redline" />
      </Tabs.Content>
    </Properties.Tabs>
  );
};

export const TextForm = (): ReactElement => {
  const theme = Theming.use();
  return (
    <Form.Sections x>
      <Form.Section title="Text">
        <Form.TextField path="value" label="Text" padHelpText={false} />
        <Form.Field<text.Level>
          path="level"
          label="Size"
          hideIfNull
          padHelpText={false}
        >
          {(p) => <Select.Text.Level {...p} />}
        </Form.Field>
        <Form.Field<text.Weight> path="weight" label="Weight" padHelpText={false}>
          {(p) => <Select.Text.Weight {...p} />}
        </Form.Field>
        <Form.Field<Flex.Alignment>
          path="align"
          label="Alignment"
          hideIfNull
          padHelpText={false}
        >
          {(p) => <Select.Flex.Alignment {...p} />}
        </Form.Field>
      </Form.Section>
      <Form.Section title="Appearance">
        <Color.Field path="textColor" label="Text" fallback={theme.colors.gray.l11} />
        <Color.Field path="fillColor" label="Fill" fallback={color.ZERO} />
      </Form.Section>
    </Form.Sections>
  );
};
