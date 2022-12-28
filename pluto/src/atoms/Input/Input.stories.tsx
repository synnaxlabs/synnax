import { useState } from "react";

import type { ComponentMeta } from "@storybook/react";
import { TimeStamp } from "@synnaxlabs/client";

import { Input } from ".";

const story: ComponentMeta<typeof Input> = {
  title: "Atoms/Input",
  component: Input,
};

export const Basic = (): JSX.Element => {
  const [value, setValue] = useState("");
  return <Input value={value} onChange={setValue} />;
};

export const Time = (): JSX.Element => {
  const [value, setValue] = useState(TimeStamp.now().valueOf());
  return <Input.Time value={value} onChange={setValue} />;
};

export const Date = (): JSX.Element => {
  const [value, setValue] = useState(TimeStamp.now().valueOf());
  return <Input.Date value={value} onChange={setValue} />;
};

export default story;
