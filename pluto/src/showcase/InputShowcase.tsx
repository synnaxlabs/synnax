// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Optional } from "@synnaxlabs/x";
import { useState } from "react";

import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Input } from "@/input";

import { PADDING_STYLE } from "./constants";

export interface InputShowcaseTextProps
  extends Optional<Input.TextProps, "value" | "onChange"> {}

export const InputShowcaseText = (props: InputShowcaseTextProps) => {
  const [value, setValue] = useState("");
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

const INPUT_PLACEHOLDER = (
  <>
    <Icon.Search />
    Catalyst
  </>
);

export const InputShowcase = () => (
  <Flex.Box y style={PADDING_STYLE} bordered rounded={1}>
    <Flex.Box x>
      <Flex.Box y>
        <InputShowcaseText placeholder="Catalyst" size="huge" />
        <InputShowcaseText placeholder="Catalyst" size="large" />
        <InputShowcaseText placeholder="Catalyst" size="medium" />
        <InputShowcaseText placeholder="Catalyst" size="small" />
        <InputShowcaseText placeholder="Catalyst" size="tiny" />
      </Flex.Box>
      <Flex.Box y>
        <InputShowcaseText placeholder="Catalyst" size="huge" variant="text" />
        <InputShowcaseText placeholder="Catalyst" size="large" variant="text" />
        <InputShowcaseText placeholder="Catalyst" size="medium" variant="text" />
        <InputShowcaseText placeholder="Catalyst" size="small" variant="text" />
        <InputShowcaseText placeholder="Catalyst" size="tiny" variant="text" />
      </Flex.Box>
      <Flex.Box y>
        <InputShowcaseText placeholder={INPUT_PLACEHOLDER} size="huge" />
        <InputShowcaseText placeholder={INPUT_PLACEHOLDER} size="large" />
        <InputShowcaseText placeholder={INPUT_PLACEHOLDER} size="medium" />
        <InputShowcaseText placeholder={INPUT_PLACEHOLDER} size="small" />
        <InputShowcaseText placeholder={INPUT_PLACEHOLDER} size="tiny" />
      </Flex.Box>
      <Flex.Box y>
        <InputShowcaseText endContent={"m/s"} size="huge" />
        <InputShowcaseText endContent={"m/s"} size="large" />
        <InputShowcaseText endContent={"m/s"} size="medium" />
        <InputShowcaseText endContent={"m/s"} size="small" />
        <InputShowcaseText endContent={"m/s"} size="tiny" />
      </Flex.Box>
      <Flex.Box y>
        <InputShowcaseNumeric placeholder="Catalyst" size="huge" />
        <InputShowcaseNumeric placeholder="Catalyst" size="large" />
        <InputShowcaseNumeric placeholder="Catalyst" size="medium" />
        <InputShowcaseNumeric placeholder="Catalyst" size="small" />
        <InputShowcaseNumeric placeholder="Catalyst" size="tiny" />
      </Flex.Box>
      <Flex.Box y>
        <InputShowcaseNumeric placeholder="Catalyst" endContent="m/s" size="huge" />
        <InputShowcaseNumeric placeholder="Catalyst" endContent="m/s" size="large" />
        <InputShowcaseNumeric placeholder="Catalyst" endContent="m/s" size="medium" />
        <InputShowcaseNumeric placeholder="Catalyst" endContent="m/s" size="small" />
        <InputShowcaseNumeric placeholder="Catalyst" endContent="m/s" size="tiny" />
      </Flex.Box>
    </Flex.Box>
    <Flex.Box x>
      <Flex.Box y>
        <InputShowcaseText disabled placeholder="Disabled" />
        <InputShowcaseText disabled placeholder={INPUT_PLACEHOLDER} />
      </Flex.Box>
      <Flex.Box y background={1} style={PADDING_STYLE} bordered rounded={1}>
        <InputShowcaseText placeholder="Catalyst" endContent="m/s" contrast={1} />
      </Flex.Box>
      <Flex.Box y background={2} style={PADDING_STYLE} bordered rounded={1}>
        <InputShowcaseText placeholder="Catalyst" endContent="m/s" contrast={2} />
      </Flex.Box>
      <Flex.Box y background={3} style={PADDING_STYLE} bordered rounded={1}>
        <InputShowcaseText placeholder="Catalyst" contrast={3} />
      </Flex.Box>
    </Flex.Box>
    <Flex.Box x>
      <InputShowcaseSwitch value={false} />
      <InputShowcaseSwitch value={true} />
    </Flex.Box>
    <Flex.Box x gap="large">
      <Input.Item label="Catalyst" helpText="Catalyst">
        <InputShowcaseText placeholder="Catalyst" endContent="m/s" />
      </Input.Item>
      <Input.Item label="Catalyst" helpText="Catalyst" helpTextVariant="warning">
        <InputShowcaseText placeholder="Catalyst" endContent="m/s" />
      </Input.Item>
      <Input.Item label="Catalyst" helpText="Catalyst" helpTextVariant="success">
        <InputShowcaseText placeholder="Catalyst" endContent="m/s" />
      </Input.Item>
    </Flex.Box>
  </Flex.Box>
);
