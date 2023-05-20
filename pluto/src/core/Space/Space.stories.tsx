// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Meta, StoryFn } from "@storybook/react";

import { Space, SpaceProps } from ".";

import { Button } from "@/core/Button";
import { ComponentSizes } from "@/util/component";

const story: Meta<typeof Space> = {
  title: "Core/Space",
  component: Space,
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
      options: Space.Alignments,
    },
    justify: {
      control: { type: "select" },
      options: Space.Justifications,
    },
  },
};

const Template: StoryFn<typeof Space> = (args: SpaceProps) => (
  <Space {...args}>
    <Button>Button 1</Button>
    <Button>Button 1</Button>
    <Button>Button 1</Button>
  </Space>
);

export const Basic = Template.bind({});
Basic.args = {
  direction: "x",
  size: "medium",
  align: "center",
  justify: "start",
};

// eslint-disable-next-line import/no-default-export
export default story;
