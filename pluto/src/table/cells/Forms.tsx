// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/table/cells/Forms.css";

import { type channel } from "@synnaxlabs/client";
import { color, type notation, type text } from "@synnaxlabs/x";
import { type PropsWithChildren, useEffect } from "react";

import { Channel } from "@/channel";
import { Color } from "@/color";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Form } from "@/form";
import { Icon } from "@/icon";
import { Input } from "@/input";
import { Notation } from "@/notation";
import { Select } from "@/select";
import { type Variant } from "@/table/cells/registry";
import { Tabs } from "@/tabs";
import { Theming } from "@/theming";
import { Staleness } from "@/vis/staleness";
import { Value } from "@/vis/value";

export interface FormProps {
  onVariantChange: (variant: Variant) => void;
}

const ValueFormWrapper = (props: PropsWithChildren) => (
  <Flex.Box {...props} className={CSS.B("table-cell-value-form")} y />
);

interface ColorFieldProps {
  path: string;
  label: string;
  /** Color shown while the field is unset. */
  fallback: color.Crude;
}

// The cell colors are optional: an unset color renders from the theme. Form.Field
// hides an absent path, which would leave the user no way to set one, so the swatch
// reads the value directly and writes only what the user picks.
const ColorField = ({ path, label, fallback }: ColorFieldProps) => {
  const { set } = Form.useContext();
  const value = Form.useFieldValue<color.Crude>(path, { optional: true });
  return (
    <Input.Item label={label} align="start" padHelpText={false}>
      <Color.Swatch
        value={value ?? fallback}
        onChange={(next: color.Color) => set(path, next)}
        bordered
      />
    </Input.Item>
  );
};

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
      <Input.Item label="Channel" grow>
        <Channel.SelectSingle
          value={value.channel}
          onChange={(key: channel.Key | null) =>
            onChange({ ...value, channel: key ?? 0 })
          }
        />
      </Input.Item>
      <Flex.Box x>
        <Input.Item label="Notation">
          <Notation.Select
            value={value.notation}
            onChange={(next: notation.Notation) =>
              onChange({ ...value, notation: next })
            }
          />
        </Input.Item>
        <Input.Item label="Precision" align="start">
          <Input.Numeric
            value={value.precision ?? 2}
            bounds={{ lower: 0, upper: 10 }}
            onChange={(precision) => onChange({ ...value, precision })}
          />
        </Input.Item>
        <Input.Item label="Averaging window" align="start">
          <Input.Numeric
            value={value.rollingAverage}
            bounds={{ lower: 1, upper: 100 }}
            onChange={(rollingAverage) => onChange({ ...value, rollingAverage })}
          />
        </Input.Item>
        <Staleness.Fields />
      </Flex.Box>
    </>
  );
};

// A cell carries no redline until one is edited, and the bound and gradient fields
// need the subtree to exist. Opening the tab materializes it.
const RedlineForm = () => {
  const { set } = Form.useContext();
  const redline = Form.useFieldValue<Value.Redline>("redline", { optional: true });
  const absent = redline == null;
  useEffect(() => {
    if (absent) set("redline", Value.ZERO_READLINE);
  }, [absent, set]);
  if (absent) return null;
  return <Value.RedlineForm path="redline" />;
};

export const ValueForm = ({ onVariantChange }: FormProps) => {
  const theme = Theming.use();
  return (
    <Tabs.Frame initialValue="style">
      <Tabs.Selector>
        <Tabs.Tab itemKey="style">Style</Tabs.Tab>
        <Tabs.Tab itemKey="telem">Telemetry</Tabs.Tab>
        <Tabs.Tab itemKey="redline">Redline</Tabs.Tab>
      </Tabs.Selector>
      <Tabs.Content itemKey="style">
        <ValueFormWrapper>
          <Flex.Box x>
            <Input.Item label="Variant" padHelpText={false}>
              <SelectVariant onChange={onVariantChange} value="value" />
            </Input.Item>
            <ColorField path="color" label="Color" fallback={theme.colors.gray.l11} />
            <Form.Field<text.Level>
              path="level"
              label="Size"
              hideIfNull
              padHelpText={false}
            >
              {(p) => <Select.Text.Level {...p} />}
            </Form.Field>
          </Flex.Box>
        </ValueFormWrapper>
      </Tabs.Content>
      <Tabs.Content itemKey="telem">
        <ValueFormWrapper>
          <TelemForm />
        </ValueFormWrapper>
      </Tabs.Content>
      <Tabs.Content itemKey="redline">
        <ValueFormWrapper>
          <RedlineForm />
        </ValueFormWrapper>
      </Tabs.Content>
    </Tabs.Frame>
  );
};

export const TextForm = ({ onVariantChange }: FormProps) => (
  <Flex.Box x grow className={CSS.B("table-cell-text-form")}>
    <Input.Item label="Variant" padHelpText={false}>
      <SelectVariant onChange={onVariantChange} value="text" />
    </Input.Item>
    <Form.TextField path="value" label="Text" />
    <Form.Field<text.Level> path="level" label="Size" hideIfNull padHelpText={false}>
      {(p) => <Select.Text.Level {...p} />}
    </Form.Field>
    <Form.Field<text.Weight> path="weight" label="Weight" padHelpText={false}>
      {(p) => <Select.Text.Weight {...p} />}
    </Form.Field>
    <Form.Field<Flex.Alignment> path="align" label="Alignment" hideIfNull>
      {(p) => <Select.Flex.Alignment {...p} />}
    </Form.Field>
    <ColorField path="backgroundColor" label="Background" fallback={color.ZERO} />
  </Flex.Box>
);

const VARIANT_DATA: Select.StaticEntry<Variant>[] = [
  { key: "text", name: "Text", icon: <Icon.Text /> },
  { key: "value", name: "Value", icon: <Icon.Channel /> },
];

export interface SelectVariantProps extends Omit<
  Select.StaticProps<Variant>,
  "data" | "resourceName"
> {}

export const SelectVariant = ({ className, ...rest }: SelectVariantProps) => (
  <Select.Static
    {...rest}
    className={CSS.cls(CSS.B("table-cell-select-variant"), className)}
    data={VARIANT_DATA}
    resourceName="variant"
  />
);
