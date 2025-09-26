// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Optional, TimeStamp } from "@synnaxlabs/x";
import { useState } from "react";

import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Input } from "@/input";
import { Text } from "@/text";

import { SubcategorySection } from "./SubcategorySection";

export interface InputShowcaseTextProps
  extends Optional<Input.TextProps, "value" | "onChange"> {}

export const InputShowcaseText = (props: InputShowcaseTextProps) => {
  const [value, setValue] = useState<string>(props.value ?? "");
  return <Input.Text {...props} value={value} onChange={setValue} />;
};

export interface InputShowcaseNumericProps
  extends Optional<Input.NumericProps, "value" | "onChange"> {}

export const InputShowcaseNumeric = (props: InputShowcaseNumericProps) => {
  const [value, setValue] = useState(0);
  return <Input.Numeric {...props} value={value} onChange={setValue} />;
};

export interface InputShowcaseSwitchProps
  extends Optional<Input.SwitchProps, "value" | "onChange"> {}

export const InputShowcaseSwitch = (props: InputShowcaseSwitchProps) => {
  const [value, setValue] = useState(props.value ?? false);
  return <Input.Switch {...props} value={value} onChange={setValue} />;
};

export interface InputShowcaseCheckboxProps
  extends Optional<Input.CheckboxProps, "value" | "onChange"> {}

export const InputShowcaseCheckbox = (props: InputShowcaseCheckboxProps) => {
  const [value, setValue] = useState(props.value ?? false);
  return <Input.Checkbox {...props} value={value} onChange={setValue} />;
};

export interface InputShowcaseDateTimeProps
  extends Optional<Input.DateTimeProps, "value" | "onChange"> {}

export const InputShowcaseDateTime = (props: InputShowcaseDateTimeProps) => {
  const [value, setValue] = useState(Number(TimeStamp.now().valueOf()));
  return <Input.DateTime {...props} value={value} onChange={setValue} />;
};

const INPUT_PLACEHOLDER = (
  <>
    <Icon.Search />
    Catalyst
  </>
);

export interface InputShowcaseTextAreaProps
  extends Optional<Input.TextProps, "value" | "onChange"> {}

export const InputShowcaseTextArea = (props: InputShowcaseTextAreaProps) => {
  const [value, setValue] = useState("");
  return <Input.Text {...props} value={value} onChange={setValue} area />;
};

export const InputShowcase = () => (
  <Flex.Box y pack empty>
    <Flex.Box x pack grow sharp>
      <SubcategorySection
        title="Text Input Sizes & Variants"
        description="Text inputs in different sizes (huge, large, medium, small, tiny) with standard, shadow, text, and preview variants"
      >
        <Flex.Box x gap="large">
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Standard
            </Text.Text>
            <InputShowcaseText placeholder="Catalyst" size="huge" />
            <InputShowcaseText placeholder="Catalyst" size="large" />
            <InputShowcaseText placeholder="Catalyst" size="medium" />
            <InputShowcaseText placeholder="Catalyst" size="small" />
            <InputShowcaseText placeholder="Catalyst" size="tiny" />
          </Flex.Box>
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Shadow Variant
            </Text.Text>
            <InputShowcaseText placeholder="Catalyst" size="huge" variant="shadow" />
            <InputShowcaseText placeholder="Catalyst" size="large" variant="shadow" />
            <InputShowcaseText placeholder="Catalyst" size="medium" variant="shadow" />
            <InputShowcaseText placeholder="Catalyst" size="small" variant="shadow" />
            <InputShowcaseText placeholder="Catalyst" size="tiny" variant="shadow" />
          </Flex.Box>
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Text Variant
            </Text.Text>
            <InputShowcaseText placeholder="Catalyst" size="huge" variant="text" />
            <InputShowcaseText placeholder="Catalyst" size="large" variant="text" />
            <InputShowcaseText placeholder="Catalyst" size="medium" variant="text" />
            <InputShowcaseText placeholder="Catalyst" size="small" variant="text" />
            <InputShowcaseText placeholder="Catalyst" size="tiny" variant="text" />
          </Flex.Box>
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Preview Variant
            </Text.Text>
            <InputShowcaseText size="huge" variant="preview" value="Catalyst" />
            <InputShowcaseText size="large" variant="preview" value="Catalyst" />
            <InputShowcaseText size="medium" variant="preview" value="Catalyst" />
            <InputShowcaseText size="small" variant="preview" value="Catalyst" />
            <InputShowcaseText size="tiny" variant="preview" value="Catalyst" />
          </Flex.Box>
        </Flex.Box>
      </SubcategorySection>

      <SubcategorySection
        title="Enhanced Text Inputs"
        description="Text inputs with icons, end content units, and rich placeholder content"
      >
        <Flex.Box x gap="large">
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              With Icon Placeholder
            </Text.Text>
            <InputShowcaseText placeholder={INPUT_PLACEHOLDER} size="huge" />
            <InputShowcaseText placeholder={INPUT_PLACEHOLDER} size="large" />
            <InputShowcaseText placeholder={INPUT_PLACEHOLDER} size="medium" />
            <InputShowcaseText placeholder={INPUT_PLACEHOLDER} size="small" />
            <InputShowcaseText placeholder={INPUT_PLACEHOLDER} size="tiny" />
          </Flex.Box>
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              With End Content
            </Text.Text>
            <InputShowcaseText endContent="m/s" size="huge" />
            <InputShowcaseText endContent="m/s" size="large" />
            <InputShowcaseText endContent="m/s" size="medium" />
            <InputShowcaseText endContent="m/s" size="small" />
            <InputShowcaseText endContent="m/s" size="tiny" />
          </Flex.Box>
        </Flex.Box>
      </SubcategorySection>
    </Flex.Box>

    <Flex.Box x pack grow sharp>
      <SubcategorySection
        title="Numeric Inputs"
        description="Numeric inputs for number values with optional units and formatting"
      >
        <Flex.Box x gap="large">
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Standard Numeric
            </Text.Text>
            <InputShowcaseNumeric placeholder="Catalyst" size="huge" />
            <InputShowcaseNumeric placeholder="Catalyst" size="large" />
            <InputShowcaseNumeric placeholder="Catalyst" size="medium" />
            <InputShowcaseNumeric placeholder="Catalyst" size="small" />
            <InputShowcaseNumeric placeholder="Catalyst" size="tiny" />
          </Flex.Box>
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              With Units
            </Text.Text>
            <InputShowcaseNumeric placeholder="Catalyst" endContent="m/s" size="huge" />
            <InputShowcaseNumeric
              placeholder="Catalyst"
              endContent="m/s"
              size="large"
            />
            <InputShowcaseNumeric
              placeholder="Catalyst"
              endContent="m/s"
              size="medium"
            />
            <InputShowcaseNumeric
              placeholder="Catalyst"
              endContent="m/s"
              size="small"
            />
            <InputShowcaseNumeric placeholder="Catalyst" endContent="m/s" size="tiny" />
          </Flex.Box>
        </Flex.Box>
      </SubcategorySection>

      <SubcategorySection
        title="Switch Controls"
        description="Toggle switches for boolean values in on and off states"
      >
        <Flex.Box y gap="medium">
          <Text.Text level="small" weight={500}>
            Switch States
          </Text.Text>
          <Flex.Box x gap="medium">
            <Flex.Box y gap="small" align="center">
              <Text.Text level="small">Off</Text.Text>
              <InputShowcaseSwitch value={false} />
            </Flex.Box>
            <Flex.Box y gap="small" align="center">
              <Text.Text level="small">On</Text.Text>
              <InputShowcaseSwitch value />
            </Flex.Box>
          </Flex.Box>
          <Text.Text level="small" weight={500}>
            Preview Variant
          </Text.Text>
          <Flex.Box x gap="medium">
            <Flex.Box y gap="small" align="center">
              <Text.Text level="small">False</Text.Text>
              <InputShowcaseSwitch value={false} variant="preview" />
            </Flex.Box>
            <Flex.Box y gap="small" align="center">
              <Text.Text level="small">True</Text.Text>
              <InputShowcaseSwitch value variant="preview" />
            </Flex.Box>
          </Flex.Box>
        </Flex.Box>
      </SubcategorySection>

      <SubcategorySection
        title="Checkbox Controls"
        description="Checkbox inputs for boolean selection with different sizes and states"
      >
        <Flex.Box y gap="medium">
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Checkbox Sizes
            </Text.Text>
            <Flex.Box x gap="medium" align="center">
              <Flex.Box y gap="small" align="center">
                <Text.Text level="small">Huge</Text.Text>
                <InputShowcaseCheckbox size="huge" />
              </Flex.Box>
              <Flex.Box y gap="small" align="center">
                <Text.Text level="small">Large</Text.Text>
                <InputShowcaseCheckbox size="large" />
              </Flex.Box>
              <Flex.Box y gap="small" align="center">
                <Text.Text level="small">Medium</Text.Text>
                <InputShowcaseCheckbox size="medium" />
              </Flex.Box>
              <Flex.Box y gap="small" align="center">
                <Text.Text level="small">Small</Text.Text>
                <InputShowcaseCheckbox size="small" />
              </Flex.Box>
              <Flex.Box y gap="small" align="center">
                <Text.Text level="small">Tiny</Text.Text>
                <InputShowcaseCheckbox size="tiny" />
              </Flex.Box>
            </Flex.Box>
          </Flex.Box>
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Checkbox States
            </Text.Text>
            <Flex.Box x gap="medium" align="center">
              <Flex.Box y gap="small" align="center">
                <Text.Text level="small">Unchecked</Text.Text>
                <InputShowcaseCheckbox value={false} />
              </Flex.Box>
              <Flex.Box y gap="small" align="center">
                <Text.Text level="small">Checked</Text.Text>
                <InputShowcaseCheckbox value />
              </Flex.Box>
              <Flex.Box y gap="small" align="center">
                <Text.Text level="small">Disabled</Text.Text>
                <InputShowcaseCheckbox disabled value={false} />
              </Flex.Box>
            </Flex.Box>
          </Flex.Box>
          <Flex.Box y gap="small">
            <Text.Text level="small" weight={500}>
              Preview Variant
            </Text.Text>
            <Flex.Box x gap="medium" align="center">
              <Flex.Box y gap="small" align="center">
                <Text.Text level="small">False</Text.Text>
                <InputShowcaseCheckbox value={false} variant="preview" />
              </Flex.Box>
              <Flex.Box y gap="small" align="center">
                <Text.Text level="small">True</Text.Text>
                <InputShowcaseCheckbox value variant="preview" />
              </Flex.Box>
            </Flex.Box>
          </Flex.Box>
        </Flex.Box>
      </SubcategorySection>
    </Flex.Box>

    <SubcategorySection
      title="Input States & Background Contrast"
      description="Disabled inputs and inputs on different background contrast levels"
    >
      <Flex.Box y gap="medium">
        <Flex.Box y gap="small">
          <Text.Text level="small" weight={500}>
            Disabled State
          </Text.Text>
          <Flex.Box x gap="medium">
            <InputShowcaseText disabled placeholder="Disabled" />
            <InputShowcaseText disabled placeholder={INPUT_PLACEHOLDER} />
          </Flex.Box>
        </Flex.Box>
        <Flex.Box y gap="small">
          <Text.Text level="small" weight={500}>
            Background Contrast
          </Text.Text>
          <Flex.Box x gap="medium">
            <Flex.Box y background={1} style={{ padding: "2rem" }} bordered rounded={1}>
              <Text.Text level="small" weight={500} style={{ marginBottom: "1rem" }}>
                Level 1
              </Text.Text>
              <InputShowcaseText placeholder="Catalyst" endContent="m/s" contrast={1} />
            </Flex.Box>
            <Flex.Box y background={2} style={{ padding: "2rem" }} bordered rounded={1}>
              <Text.Text level="small" weight={500} style={{ marginBottom: "1rem" }}>
                Level 2
              </Text.Text>
              <InputShowcaseText placeholder="Catalyst" endContent="m/s" contrast={2} />
            </Flex.Box>
            <Flex.Box y background={3} style={{ padding: "2rem" }} bordered rounded={1}>
              <Text.Text level="small" weight={500} style={{ marginBottom: "1rem" }}>
                Level 3
              </Text.Text>
              <InputShowcaseText placeholder="Catalyst" contrast={3} />
            </Flex.Box>
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>
    </SubcategorySection>

    <SubcategorySection
      title="Custom Colors"
      description="Inputs with custom colors and different background contrast levels"
    >
      <Flex.Box x gap="large">
        <InputShowcaseText placeholder="Catalyst" color="#00FF00" />
        <InputShowcaseNumeric placeholder="Catalyst" color="#00FF00" />
      </Flex.Box>
    </SubcategorySection>

    <SubcategorySection
      title="Form Items with Labels & Help Text"
      description="Input components wrapped in Item containers with labels and help text in different states"
    >
      <Flex.Box x gap="large">
        <Input.Item label="Catalyst" helpText="Catalyst" status="error">
          <InputShowcaseText placeholder="Catalyst" endContent="m/s" status="error" />
        </Input.Item>
        <Input.Item label="Catalyst" helpText="Catalyst" status="warning">
          <InputShowcaseText placeholder="Catalyst" endContent="m/s" status="warning" />
        </Input.Item>
        <Input.Item label="Catalyst" helpText="Catalyst" status="success">
          <InputShowcaseText placeholder="Catalyst" endContent="m/s" status="success" />
        </Input.Item>
      </Flex.Box>
    </SubcategorySection>

    <SubcategorySection
      title="Text Area"
      description="Text area component with different sizes and variants"
    >
      <Flex.Box x gap="large">
        <InputShowcaseText placeholder="Catalyst" area />
      </Flex.Box>
    </SubcategorySection>

    <SubcategorySection
      title="Text Area"
      description="Text area component with different sizes and variants"
    >
      <Flex.Box x gap="large">
        <InputShowcaseDateTime placeholder="Catalyst" />
      </Flex.Box>
    </SubcategorySection>
  </Flex.Box>
);
