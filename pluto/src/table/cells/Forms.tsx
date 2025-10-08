// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { type PropsWithChildren, useCallback } from "react";

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

const valueFormStyle = { padding: "2rem" };

const ValueFormWrapper = (props: PropsWithChildren) => (
  <Flex.Box {...props} style={valueFormStyle} y />
);

const valueTabs = [
  { tabKey: "style", name: "Style" },
  { tabKey: "telem", name: "Telemetry" },
  { tabKey: "redline", name: "Redline" },
];

export const ValueForm = ({ onVariantChange }: FormProps) => {
  const content: Tabs.RenderProp = useCallback(({ tabKey }) => {
    switch (tabKey) {
      case "telem":
        return (
          <ValueFormWrapper>
            <Value.TelemForm path="" />
          </ValueFormWrapper>
        );
      case "redline":
        return (
          <ValueFormWrapper>
            <Value.RedlineForm path="redline" />
          </ValueFormWrapper>
        );
      default:
        return (
          <ValueFormWrapper>
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
                {({ value, onChange, variant: _, ...rest }) => (
                  <Select.Text.Level value={value} onChange={onChange} {...rest} />
                )}
              </Form.Field>
            </Flex.Box>
          </ValueFormWrapper>
        );
    }
  }, []);
  const tabsProps = Tabs.useStatic({ tabs: valueTabs, content });
  return <Tabs.Tabs {...tabsProps} />;
};

const textFormStyle = { padding: "2rem" };

export const TextForm = ({ onVariantChange }: FormProps) => (
  <Flex.Box x grow style={textFormStyle}>
    <Input.Item label="Variant" padHelpText={false}>
      <SelectVariant onChange={onVariantChange} value="text" />
    </Input.Item>
    <Form.TextField path="value" label="Text" />
    <Form.Field<Text.Level> path="level" label="Size" hideIfNull padHelpText={false}>
      {({ value, onChange, variant: _, ...rest }) => (
        <Select.Text.Level value={value} onChange={onChange} {...rest} />
      )}
    </Form.Field>
    <Form.Field<Text.Weight> path="weight" label="Weight" padHelpText={false}>
      {({ value, onChange, variant: _, ...rest }) => (
        <Select.Text.Weight value={value} onChange={onChange} {...rest} />
      )}
    </Form.Field>
    <Form.Field<Flex.Alignment> path="align" label="Alignment" hideIfNull>
      {({ value, onChange, variant: _, ...rest }) => (
        <Select.Flex.Alignment value={value} onChange={onChange} {...rest} />
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

const VARIANT_DATA: Select.StaticEntry<Variant>[] = [
  { key: "text", name: "Text", icon: <Icon.Text /> },
  { key: "value", name: "Value", icon: <Icon.Channel /> },
];

export interface SelectVariantProps
  extends Omit<Select.StaticProps<Variant>, "data" | "resourceName"> {}

export const SelectVariant = (props: SelectVariantProps) => (
  <Select.Static {...props} data={VARIANT_DATA} resourceName="Variant" />
);
