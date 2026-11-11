// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type table } from "@synnaxlabs/client";
import { Color } from "@synnaxlabs/lyra/color";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Tabs } from "@synnaxlabs/lyra/tabs";
import { Theming } from "@synnaxlabs/lyra/theming";
import { color, deep, type notation } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Channel } from "@/channel";
import { Notation } from "@/notation";
import { Properties } from "@/vis/properties";
import { Staleness } from "@/vis/staleness";
import { Value } from "@/vis/value";

const path = deep.path<table.ValueCellConfig>();
const textPath = deep.path<table.TextCellConfig>();

/** The colors a cell paints while one of its colors is absent. */
export const colorFallbacks = {
  textColor: (theme: Theming.Theme): color.Color => theme.colors.gray.l11,
  fillColor: (): color.Color => color.ZERO,
};

/** Props for a cell variant's form. */
export interface FormProps extends Properties.SelectionProps {}

const TelemForm = () => {
  const { value, onChange } = Form.useField(path);
  return (
    <>
      <Form.Section title="Source">
        <Input.Item label="Channel" padHelpText={false}>
          <Channel.SelectSingle
            allowNone
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
          <Notation.Precision
            value={value.precision ?? 2}
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
              path={path.textColor}
              label="Text"
              fallback={colorFallbacks.textColor(theme)}
            />
            <Color.Field
              path={path.fillColor}
              label="Fill"
              fallback={colorFallbacks.fillColor()}
            />
            <Form.Field path={path.level} label="Size" padHelpText={false}>
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
        <Value.RedlineForm path={path.redline} />
      </Tabs.Content>
    </Properties.Tabs>
  );
};

export const TextForm = (): ReactElement => {
  const theme = Theming.use();
  return (
    <Form.Sections x>
      <Form.Section title="Text">
        <Form.TextField path={textPath.value} label="Text" padHelpText={false} />
        <Form.Field path={textPath.level} label="Size" padHelpText={false}>
          {(p) => <Select.Text.Level {...p} />}
        </Form.Field>
        <Form.Field path={textPath.weight} label="Weight" padHelpText={false}>
          {(p) => <Select.Text.Weight {...p} />}
        </Form.Field>
        <Form.Field path={textPath.align} label="Alignment" padHelpText={false}>
          {(p) => <Select.Flex.Alignment {...p} />}
        </Form.Field>
      </Form.Section>
      <Form.Section title="Appearance">
        <Color.Field
          path={textPath.textColor}
          label="Text"
          fallback={colorFallbacks.textColor(theme)}
        />
        <Color.Field
          path={textPath.fillColor}
          label="Fill"
          fallback={colorFallbacks.fillColor()}
        />
      </Form.Section>
    </Form.Sections>
  );
};
