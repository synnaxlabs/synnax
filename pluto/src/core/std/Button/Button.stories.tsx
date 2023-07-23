// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useState } from "react";

import type { Meta, StoryFn } from "@storybook/react";
import { AiOutlineDelete } from "react-icons/ai";

import { Button, ButtonProps } from "@/core/std/Button";

const story: Meta<typeof Button> = {
  title: "Core/Standard/Button",
  component: Button,
  argTypes: {
    variant: {
      options: ["filled", "outlined", "text"],
      control: { type: "select" },
    },
  },
};

const Template = (args: ButtonProps): ReactElement => <Button {...args} />;

export const Primary: StoryFn<typeof Button> = Template.bind({});
Primary.args = {
  size: "medium",
  startIcon: <AiOutlineDelete />,
  children: "Button",
};

export const Outlined = (): ReactElement => (
  <Button variant="outlined" endIcon={<AiOutlineDelete />}>
    Button
  </Button>
);

export const Toggle = (): ReactElement => {
  const [value, setValue] = useState(false);
  return (
    <Button.ToggleIcon value={value} onChange={() => setValue((c) => !c)}>
      <AiOutlineDelete />
    </Button.ToggleIcon>
  );
};

export const IconOnly = (): ReactElement => (
  <Button.Icon>
    <AiOutlineDelete />
  </Button.Icon>
);

// eslint-disable-next-line import/no-default-export
export default story;
