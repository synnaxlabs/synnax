// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useState } from "react";

import type { ComponentMeta, ComponentStory } from "@storybook/react";
import { AiOutlineDelete } from "react-icons/ai";

import { Button, ButtonProps } from ".";

const story: ComponentMeta<typeof Button> = {
  title: "Core/Button",
  component: Button,
  argTypes: {
    variant: {
      options: ["filled", "outlined", "text"],
      control: { type: "select" },
    },
  },
};

const Template = (args: ButtonProps): JSX.Element => <Button {...args} />;

export const Primary: ComponentStory<typeof Button> = Template.bind({});
Primary.args = {
  size: "medium",
  startIcon: <AiOutlineDelete />,
  children: "Button",
};

export const Outlined = (): JSX.Element => (
  <Button variant="outlined" endIcon={<AiOutlineDelete />}>
    Button
  </Button>
);

export const Toggle = (): JSX.Element => {
  const [value, setValue] = useState(false);
  return (
    <Button.ToggleIcon value={value} onChange={() => setValue((c) => !c)}>
      <AiOutlineDelete />
    </Button.ToggleIcon>
  );
};

export const IconOnly = (): JSX.Element => (
  <Button.Icon>
    <AiOutlineDelete />
  </Button.Icon>
);

// eslint-disable-next-line import/no-default-export
export default story;
