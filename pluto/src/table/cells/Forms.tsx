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
import { CSS } from "@synnaxlabs/lyra/css";
import { type Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Tabs } from "@synnaxlabs/lyra/tabs";
import { Theming } from "@synnaxlabs/lyra/theming";
import { color, type notation, type text } from "@synnaxlabs/x";

import { Channel } from "@/channel";
import { Color } from "@/color";
import { Notation } from "@/notation";
import { type Variant } from "@/table/cells/registry";
import { Staleness } from "@/vis/staleness";
import { Value } from "@/vis/value";

export interface FormProps {
  onVariantChange: (variant: Variant) => void;
}

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
    <Input.Item label={label} padHelpText={false}>
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

export const ValueForm = ({ onVariantChange }: FormProps) => {
  const theme = Theming.use();
  return (
    <Tabs.Frame initialValue="style" className={CSS.B("table-cell-value-form-tabs")}>
      <Tabs.Selector>
        <Tabs.Tab itemKey="style">Style</Tabs.Tab>
        <Tabs.Tab itemKey="telem">Telemetry</Tabs.Tab>
        <Tabs.Tab itemKey="redline">Redline</Tabs.Tab>
      </Tabs.Selector>
      <Tabs.Content itemKey="style">
        <Form.Sections x>
          <Form.Section title="Cell">
            <Input.Item label="Variant" padHelpText={false}>
              <SelectVariant onChange={onVariantChange} value="value" />
            </Input.Item>
          </Form.Section>
          <Form.Section title="Appearance">
            <ColorField path="color" label="Color" fallback={theme.colors.gray.l11} />
            <ColorField
              path="backgroundColor"
              label="Background"
              fallback={color.ZERO}
            />
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
      <Tabs.Content itemKey="telem">
        <Form.Sections x>
          <TelemForm />
        </Form.Sections>
      </Tabs.Content>
      <Tabs.Content itemKey="redline">
        <Value.RedlineForm path="redline" />
      </Tabs.Content>
    </Tabs.Frame>
  );
};

export const TextForm = ({ onVariantChange }: FormProps) => (
  <Form.Sections x>
    <Form.Section title="Cell">
      <Input.Item label="Variant" padHelpText={false}>
        <SelectVariant onChange={onVariantChange} value="text" />
      </Input.Item>
    </Form.Section>
    <Form.Section title="Text">
      <Form.TextField path="value" label="Text" padHelpText={false} />
      <Form.Field<text.Level> path="level" label="Size" hideIfNull padHelpText={false}>
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
      <ColorField path="backgroundColor" label="Background" fallback={color.ZERO} />
    </Form.Section>
  </Form.Sections>
);

export interface SelectVariantProps extends Omit<
  Select.SingleSimpleProps<Variant>,
  "children" | "resourceName"
> {}

export const SelectVariant = ({ className, ...rest }: SelectVariantProps) => (
  <Select.Simple<Variant>
    {...rest}
    className={CSS.cls(CSS.B("table-cell-select-variant"), className)}
    resourceName="variant"
  >
    <Select.Item itemKey="text">
      <Icon.Text />
      Text
    </Select.Item>
    <Select.Item itemKey="value">
      <Icon.Channel />
      Value
    </Select.Item>
  </Select.Simple>
);
