// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Meta, StoryFn } from "@storybook/react";

import { Align } from "@/align";
import { Button } from "@/button";
import { ComponentSizes } from "@/util/component";

const story: Meta<typeof Align.Space> = {
  title: "Core/Standard/Space",
  component: Align.Space,
  argTypes: {
    direction: {
      control: { type: "select" },
      options: ["x", "y"],
    },
    size: {
      control: { type: "select" },
      options: ComponentSizes,
    },
    align: {
      control: { type: "select" },
      options: Align.ALIGNMENTS,
    },
    justify: {
      control: { type: "select" },
      options: Align.JUSTIFICATIONS,
    },
  },
};

const Template: StoryFn<typeof Align.Space> = (args: Align.SpaceProps) => (
  <Align.Space {...args}>
    <Button.Button>Button 1</Button.Button>
    <Button.Button>Button 1</Button.Button>
    <Button.Button>Button 1</Button.Button>
  </Align.Space>
);

export const Basic: StoryFn = Template.bind({});
Basic.args = {
  direction: "y",
  size: "medium",
  align: "center",
  justify: "start",
};

// eslint-disable-next-line import/no-default-export
export default story;
