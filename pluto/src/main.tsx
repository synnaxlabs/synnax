// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/index.css";
import "@/main.css";

import { type channel } from "@synnaxlabs/client";
import { caseconv, type Optional } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";
import ReactDOM from "react-dom/client";

import { Button } from "@/button";
import { Channel } from "@/channel";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Input } from "@/input";
import { Pluto } from "@/pluto";
import { Select } from "@/select";
import { state } from "@/state";
import { Tag } from "@/tag";
import { Text } from "@/text";

const PADDING_STYLE = { padding: "5rem" };

const TextShowcase = () => (
  <Flex.Box x bordered style={PADDING_STYLE} rounded={1}>
    <Flex.Box y>
      <Text.Text level="h1">Hello</Text.Text>
      <Text.Text level="h2">Hello</Text.Text>
      <Text.Text level="h3">Hello</Text.Text>
      <Text.Text level="h4">Hello</Text.Text>
      <Text.Text level="h5">Hello</Text.Text>
      <Text.Text level="p">Hello</Text.Text>
      <Text.Text level="small">Hello</Text.Text>
    </Flex.Box>
    <Flex.Box y>
      <Text.Text color={11}>Hello</Text.Text>
      <Text.Text color={10}>Hello</Text.Text>
      <Text.Text color={9}>Hello</Text.Text>
      <Text.Text color={8}>Hello</Text.Text>
      <Text.Text color={7}>Hello</Text.Text>
      <Text.Text color={6}>Hello</Text.Text>
      <Text.Text color={5}>Hello</Text.Text>
      <Text.Text color={4}>Hello</Text.Text>
      <Text.Text color={3}>Hello</Text.Text>
      <Text.Text color={2}>Hello</Text.Text>
      <Text.Text color={1}>Hello</Text.Text>
    </Flex.Box>
  </Flex.Box>
);

const ButtonShowcase = () => (
  <Flex.Box y style={PADDING_STYLE} bordered rounded={1}>
    <Flex.Box x>
      <Flex.Box y>
        <Button.Button size="huge">Hello</Button.Button>
        <Button.Button size="large">Hello</Button.Button>
        <Button.Button size="medium">Hello</Button.Button>
        <Button.Button size="small">Hello</Button.Button>
        <Button.Button size="tiny">Hello</Button.Button>
      </Flex.Box>
      <Flex.Box y>
        <Button.Button size="huge" variant="filled">
          Hello
        </Button.Button>
        <Button.Button size="large" variant="filled">
          Hello
        </Button.Button>
        <Button.Button size="medium" variant="filled">
          Hello
        </Button.Button>
        <Button.Button size="small" variant="filled">
          Hello
        </Button.Button>
        <Button.Button size="tiny" variant="filled">
          Hello
        </Button.Button>
      </Flex.Box>
      <Flex.Box y>
        <Button.Button size="huge" variant="text">
          Hello
        </Button.Button>
        <Button.Button size="large" variant="text">
          Hello
        </Button.Button>
        <Button.Button size="medium" variant="text">
          Hello
        </Button.Button>
        <Button.Button size="small" variant="text">
          Hello
        </Button.Button>
        <Button.Button size="tiny" variant="text">
          Hello
        </Button.Button>
      </Flex.Box>
      <Flex.Box y>
        <Button.Button size="huge" variant="outlined">
          <Icon.Add />
          Hello
        </Button.Button>
        <Button.Button size="large" variant="outlined">
          <Icon.Add />
          Hello
        </Button.Button>
        <Button.Button size="medium" variant="outlined">
          <Icon.Add />
          Hello
        </Button.Button>
        <Button.Button size="small" variant="outlined">
          <Icon.Add />
          Hello
        </Button.Button>
        <Button.Button size="tiny" variant="outlined">
          <Icon.Add />
          Hello
        </Button.Button>
      </Flex.Box>
      <Flex.Box y>
        <Button.Button size="huge" variant="filled">
          <Icon.Add />
          Hello
        </Button.Button>
        <Button.Button size="large" variant="filled">
          <Icon.Add />
          Hello
        </Button.Button>
        <Button.Button size="medium" variant="filled">
          <Icon.Add />
          Hello
        </Button.Button>
        <Button.Button size="small" variant="filled">
          <Icon.Add />
          Hello
        </Button.Button>
        <Button.Button size="tiny" variant="filled">
          <Icon.Add />
          Hello
        </Button.Button>
      </Flex.Box>
      <Flex.Box y>
        <Button.Button size="huge" variant="text">
          <Icon.Add />
          Hello
        </Button.Button>
        <Button.Button size="large" variant="text">
          <Icon.Add />
          Hello
        </Button.Button>
        <Button.Button size="medium" variant="text">
          <Icon.Add />
          Hello
        </Button.Button>
        <Button.Button size="small" variant="text">
          <Icon.Add />
          Hello
        </Button.Button>
        <Button.Button size="tiny" variant="text">
          <Icon.Add />
          Hello
        </Button.Button>
      </Flex.Box>
    </Flex.Box>
    <Flex.Box x>
      <Flex.Box y>
        <Button.Button size="huge">
          <Icon.Auto />
        </Button.Button>
        <Button.Button size="large">
          <Icon.Auto />
        </Button.Button>
        <Button.Button size="medium">
          <Icon.Auto />
        </Button.Button>
        <Button.Button size="small">
          <Icon.Auto />
        </Button.Button>
        <Button.Button size="tiny">
          <Icon.Auto />
        </Button.Button>
      </Flex.Box>
      <Flex.Box y>
        <Button.Button size="huge" variant="filled">
          <Icon.Auto />
        </Button.Button>
        <Button.Button size="large" variant="filled">
          <Icon.Auto />
        </Button.Button>
        <Button.Button size="medium" variant="filled">
          <Icon.Auto />
        </Button.Button>
        <Button.Button size="small" variant="filled">
          <Icon.Auto />
        </Button.Button>
        <Button.Button size="tiny" variant="filled">
          <Icon.Auto />
        </Button.Button>
      </Flex.Box>
      <Flex.Box y>
        <Button.Button size="huge" variant="text">
          <Icon.Auto />
        </Button.Button>
        <Button.Button size="large" variant="text">
          <Icon.Auto />
        </Button.Button>
        <Button.Button size="medium" variant="text">
          <Icon.Auto />
        </Button.Button>
        <Button.Button size="small" variant="text">
          <Icon.Auto />
        </Button.Button>
        <Button.Button size="tiny" variant="text">
          <Icon.Auto />
        </Button.Button>
      </Flex.Box>
      <Flex.Box y>
        <Flex.Box x>
          <Button.Button disabled>Hello</Button.Button>
          <Button.Button disabled variant="filled">
            Hello
          </Button.Button>
          <Button.Button disabled variant="text">
            Hello
          </Button.Button>
        </Flex.Box>
        <Flex.Box x>
          <Button.Button sharp>Hello</Button.Button>
          <Button.Button loading>Hello</Button.Button>
        </Flex.Box>
      </Flex.Box>
    </Flex.Box>
    <Flex.Box x>
      <Flex.Box y background={1} style={PADDING_STYLE} bordered rounded={1}>
        <Button.Button contrast={1}>Hello</Button.Button>
      </Flex.Box>
      <Flex.Box y background={2} style={PADDING_STYLE} bordered rounded={1}>
        <Button.Button contrast={2}>Hello</Button.Button>
      </Flex.Box>
      <Flex.Box y background={3} style={PADDING_STYLE} bordered rounded={1}>
        <Button.Button contrast={3}>Hello</Button.Button>
      </Flex.Box>
    </Flex.Box>
  </Flex.Box>
);

const TagShowcase = () => (
  <Flex.Box y style={PADDING_STYLE} bordered rounded={1}>
    <Flex.Box x>
      <Flex.Box y>
        <Tag.Tag size="huge">Hello</Tag.Tag>
        <Tag.Tag size="large">Hello</Tag.Tag>
        <Tag.Tag size="medium">Hello</Tag.Tag>
        <Tag.Tag size="small">Hello</Tag.Tag>
        <Tag.Tag size="tiny">Hello</Tag.Tag>
      </Flex.Box>
      <Flex.Box y>
        <Tag.Tag size="huge" color="#00E3E2">
          Hello
        </Tag.Tag>
        <Tag.Tag size="large" color="#00E3E2">
          Hello
        </Tag.Tag>
        <Tag.Tag size="medium" color="#00E3E2">
          Hello
        </Tag.Tag>
        <Tag.Tag size="small" color="#00E3E2">
          Hello
        </Tag.Tag>
        <Tag.Tag size="tiny" color="#00E3E2">
          Hello
        </Tag.Tag>
      </Flex.Box>
      <Flex.Box y>
        <Tag.Tag size="huge" onClose={console.log}>
          Hello
        </Tag.Tag>
        <Tag.Tag size="large" onClose={console.log}>
          Hello
        </Tag.Tag>
        <Tag.Tag size="medium" onClose={console.log}>
          Hello
        </Tag.Tag>
        <Tag.Tag size="small" onClose={console.log}>
          Hello
        </Tag.Tag>
        <Tag.Tag size="tiny" onClose={console.log}>
          Hello
        </Tag.Tag>
      </Flex.Box>
    </Flex.Box>
  </Flex.Box>
);

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

const InputShowcase = () => (
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
        <InputShowcaseText placeholder="Catalyst" size="huge" variant="natural" />
        <InputShowcaseText placeholder="Catalyst" size="large" variant="natural" />
        <InputShowcaseText placeholder="Catalyst" size="medium" variant="natural" />
        <InputShowcaseText placeholder="Catalyst" size="small" variant="natural" />
        <InputShowcaseText placeholder="Catalyst" size="tiny" variant="natural" />
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

const SelectMultiple = () => {
  const [value, setValue] = useState<channel.Key[]>([]);
  return <Channel.SelectMultiple value={value} onChange={setValue} />;
};

const SelectSingle = () => {
  const [value, setValue] = useState<channel.Key | undefined>(undefined);
  return <Channel.SelectSingle value={value} onChange={setValue} />;
};

const SelectShowcase = () => (
  <Flex.Box y style={PADDING_STYLE} bordered rounded={1} fullWidth>
    <SelectMultiple />
    <SelectSingle />
  </Flex.Box>
);

const FlexShowcase = () => (
  <Flex.Box x style={PADDING_STYLE} bordered rounded={1} fullWidth>
    <Flex.Box y pack>
      <Flex.Box x pack>
        <Button.Button variant="filled">Hello</Button.Button>
        <Button.Button>Hello</Button.Button>
      </Flex.Box>
      <Flex.Box x pack>
        <Button.Button>Hello</Button.Button>
        <Button.Button variant="filled">Hello</Button.Button>
      </Flex.Box>
    </Flex.Box>
    <Flex.Box x pack>
      <Flex.Box y pack>
        <Button.Button>Hello</Button.Button>
        <Button.Button variant="filled">Hello</Button.Button>
      </Flex.Box>
      <Flex.Box y pack>
        <Button.Button variant="filled">Hello</Button.Button>
        <Button.Button>Hello</Button.Button>
      </Flex.Box>
    </Flex.Box>
    <Flex.Box y pack align="stretch">
      <Flex.Box x pack>
        <Button.Button>Hello</Button.Button>
        <Button.Button variant="filled">Hello</Button.Button>
      </Flex.Box>
      <Button.Button fullWidth justify="center">
        Hello
      </Button.Button>
    </Flex.Box>
    <Flex.Box y pack align="stretch">
      <Button.Button fullWidth>Hello</Button.Button>
      <InputShowcaseText />
    </Flex.Box>
  </Flex.Box>
);

const DISPLAY = ["text", "button", "input", "tag", "select", "flex"];

interface DisplaySelectorProps {
  display: (typeof DISPLAY)[number][];
  setDisplay: (display: (typeof DISPLAY)[number][]) => void;
}

const DisplaySelector = ({ display, setDisplay }: DisplaySelectorProps) => (
  <Select.Buttons multiple keys={DISPLAY} value={display} onChange={setDisplay}>
    {DISPLAY.map((d) => (
      <Select.Button key={d} itemKey={d}>
        {caseconv.capitalize(d)}
      </Select.Button>
    ))}
  </Select.Buttons>
);

const Canvas = () => {
  const [display, setDisplay] = state.usePersisted<(typeof DISPLAY)[number][]>(
    DISPLAY,
    "display",
  );
  return (
    <Flex.Box y gap="large" style={PADDING_STYLE}>
      <DisplaySelector display={display} setDisplay={setDisplay} />
      <Flex.Box x>
        {display.includes("text") && <TextShowcase />}
        {display.includes("button") && <ButtonShowcase />}
        {display.includes("tag") && <TagShowcase />}
      </Flex.Box>
      <Flex.Box x>{display.includes("input") && <InputShowcase />}</Flex.Box>
      <Flex.Box x>{display.includes("select") && <SelectShowcase />}</Flex.Box>
      <Flex.Box x>{display.includes("flex") && <FlexShowcase />}</Flex.Box>
    </Flex.Box>
  );
};

const Main = (): ReactElement => (
  <Pluto.Provider
    connParams={{
      host: "localhost",
      port: 9090,
      secure: false,
      username: "synnax",
      password: "seldon",
    }}
  >
    <Canvas />
  </Pluto.Provider>
);

ReactDOM.createRoot(document.getElementById("root")!).render(<Main />);
