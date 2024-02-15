// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useState } from "react";

import type { Meta } from "@storybook/react";
import { AiOutlineDelete } from "react-icons/ai";

import { Align } from "@/align";
import { Button } from "@/button";

const story: Meta<typeof Button.Button> = {
  title: "Button",
  component: Button.Button,
  argTypes: {
    variant: {
      options: ["filled", "outlined", "text"],
      control: { type: "select" },
    },
  },
};

export const Primary = (args: Button.ButtonProps): ReactElement => (
  <Align.Space align="start">
    <Button.Button startIcon={<AiOutlineDelete />} {...args} size="small">
      Button
    </Button.Button>
    <Button.Button startIcon={<AiOutlineDelete />} {...args} size="medium">
      Button
    </Button.Button>
    <Button.Button startIcon={<AiOutlineDelete />} {...args} size="large">
      Button
    </Button.Button>
    <Button.Button startIcon={<AiOutlineDelete />} {...args} size="large" disabled>
      Button
    </Button.Button>
    <Button.Button {...args} size="small" variant="outlined">
      Button
    </Button.Button>
    <Button.Button
      startIcon={<AiOutlineDelete />}
      {...args}
      size="medium"
      variant="outlined"
    >
      Button
    </Button.Button>
    <Button.Button
      startIcon={<AiOutlineDelete />}
      {...args}
      size="large"
      variant="outlined"
    >
      Button
    </Button.Button>
    <Button.Button
      startIcon={<AiOutlineDelete />}
      {...args}
      size="large"
      variant="outlined"
      disabled
    >
      Button
    </Button.Button>
    <Button.Button
      startIcon={<AiOutlineDelete />}
      {...args}
      size="small"
      variant="text"
    >
      Button
    </Button.Button>
    <Button.Button
      startIcon={<AiOutlineDelete />}
      {...args}
      size="medium"
      variant="text"
    >
      Button
    </Button.Button>
    <Button.Button
      startIcon={<AiOutlineDelete />}
      {...args}
      size="large"
      variant="text"
    >
      Button
    </Button.Button>
    <Button.Button
      startIcon={<AiOutlineDelete />}
      {...args}
      size="large"
      variant="text"
      disabled
    >
      Button
    </Button.Button>
  </Align.Space>
);

export const Outlined = (): ReactElement => (
  <Button.Button variant="outlined" endIcon={<AiOutlineDelete />}>
    Button
  </Button.Button>
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
