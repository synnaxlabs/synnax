// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type bounds, color, deep, scale } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { Color } from "@/color";
import { Flex } from "@/flex";
import { Form } from "@/form";
import { Icon } from "@/icon";
import { Input } from "@/input";
import { Select } from "@/select";
import { type Variant } from "@/table/cells/registry";
import { Tabs } from "@/tabs";
import { type Text } from "@/text";
import { Value } from "@/vis/value";

export interface FormProps {
  onVariantChange: (variant: Variant) => void;
}

export const ValueForm = ({ onVariantChange }: FormProps) => {
  const content: Tabs.RenderProp = useCallback(({ tabKey }) => {
    switch (tabKey) {
      case "telem":
        return (
          <Flex.Box y style={{ padding: "2rem" }}>
            <Value.TelemForm path="" />
          </Flex.Box>
        );
      case "redline":
        return (
          <Flex.Box y style={{ padding: "2rem" }}>
            <RedlineForm />
          </Flex.Box>
        );
      default:
        return (
          <Flex.Box y grow empty style={{ padding: "2rem" }}>
            <Flex.Box x>
              <Input.Item label="Variant" padHelpText={false}>
                <SelectVariant onChange={onVariantChange} value="value" />
              </Input.Item>
              <Form.Field<color.Crude>
                hideIfNull
                label="Color"
                align="start"
                padHelpText={false}
                path="color"
              >
                {({ value, onChange, variant: _, ...rest }) => (
                  <Color.Swatch
                    value={value ?? color.setAlpha(color.ZERO, 1)}
                    onChange={onChange}
                    {...rest}
                    bordered
                  />
                )}
              </Form.Field>
              <Form.Field<Text.Level>
                path="level"
                label="Size"
                hideIfNull
                padHelpText={false}
              >
                {({ value, onChange, variant: __, ...rest }) => (
                  <Select.Text.Level value={value} onChange={onChange} {...rest} />
                )}
              </Form.Field>
            </Flex.Box>
          </Flex.Box>
        );
    }
  }, []);
  const tabsProps = Tabs.useStatic({
    tabs: [
      { tabKey: "style", name: "Style" },
      { tabKey: "telem", name: "Telemetry" },
      { tabKey: "redline", name: "Redline" },
    ],
    content,
  });
  return <Tabs.Tabs {...tabsProps} size="small" />;
};

const RedlineForm = (): ReactElement => {
  const { set, get } = Form.useContext();
  const b = Form.useFieldValue<bounds.Bounds>("redline.bounds");
  const s = scale.Scale.scale<number>(0, 1).scale(b);
  return (
    <Flex.Box x grow>
      <Form.NumericField
        inputProps={{ size: "small", showDragHandle: false }}
        style={{ width: 60 }}
        label="Lower"
        path="redline.bounds.lower"
      />
      <Form.Field<color.Gradient>
        path="redline.gradient"
        label="Gradient"
        align="start"
        padHelpText={false}
      >
        {({ value, onChange }) => (
          <Color.GradientPicker
            value={deep.copy(value)}
            scale={s}
            onChange={(v) => {
              const prevB = get<bounds.Bounds>("redline.bounds").value;
              const nextBounds = { ...prevB };
              const positions = v.map((c) => c.position);
              const highestPos = s.pos(Math.max(...positions));
              const lowestPos = s.pos(Math.min(...positions));
              const highestGreater = highestPos > nextBounds.upper;
              const lowestLower = lowestPos < nextBounds.lower;
              if (highestGreater) {
                v[v.length - 1].position = 1;
                nextBounds.upper = highestPos;
              }
              if (lowestLower) {
                v[0].position = 0;
                nextBounds.lower = lowestPos;
              }
              const grad = v.map((c) => ({
                ...c,
                color: color.hex(c.color),
              }));
              if (highestGreater || lowestLower)
                set("redline", {
                  bounds: nextBounds,
                  gradient: grad,
                });
              else onChange(v.map((c) => ({ ...c, color: color.hex(c.color) })));
            }}
          />
        )}
      </Form.Field>
      <Form.NumericField
        inputProps={{ size: "small", showDragHandle: false }}
        style={{ width: 60 }}
        label="Upper"
        path="redline.bounds.upper"
      />
    </Flex.Box>
  );
};

export const TextForm = ({ onVariantChange }: FormProps) => (
  <Flex.Box x grow style={{ padding: "2rem" }}>
    <Input.Item label="Variant" padHelpText={false}>
      <SelectVariant onChange={onVariantChange} value="text" />
    </Input.Item>
    <Form.TextField path="value" label="Text" />
    <Form.Field<Text.Level> path="level" label="Size" hideIfNull padHelpText={false}>
      {({ value, onChange, variant: __, ...rest }) => (
        <Select.Text.Level value={value} onChange={onChange} {...rest} />
      )}
    </Form.Field>
    <Form.Field<Text.Weight> path="weight" label="Weight" padHelpText={false}>
      {({ value, onChange, variant: ___, ...rest }) => (
        <Select.Text.Weight value={value} onChange={onChange} {...rest} />
      )}
    </Form.Field>
    <Form.Field<Flex.Alignment> path="align" label="Alignment" hideIfNull>
      {({ value, onChange, variant: ___, ...rest }) => (
        <Flex.SelectAlignment value={value} onChange={onChange} {...rest} />
      )}
    </Form.Field>
    <Form.Field<color.Crude>
      path="backgroundColor"
      label="Background"
      align="start"
      padHelpText={false}
    >
      {({ value, onChange }) => <Color.Swatch value={value} onChange={onChange} />}
    </Form.Field>
  </Flex.Box>
);

export const VARIANT_DATA: Select.StaticEntry<Variant>[] = [
  { key: "text", name: "Text", icon: <Icon.Text /> },
  { key: "value", name: "Value", icon: <Icon.Channel /> },
];

export interface SelectVariantProps
  extends Omit<Select.StaticProps<Variant>, "data" | "resourceName"> {}

export const SelectVariant = (props: SelectVariantProps) => (
  <Select.Static data={VARIANT_DATA} {...props} resourceName="Variant" />
);
