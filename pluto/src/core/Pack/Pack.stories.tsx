// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { ComponentMeta, ComponentStory } from "@storybook/react";

import { Pack } from ".";

import { Button } from "@/core/Button";

const story: ComponentMeta<typeof Pack> = {
  title: "Core/Pack",
  component: Pack,
  argTypes: {
    direction: {
      control: { type: "select" },
      options: ["x", "y"],
    },
  },
};

const Template: ComponentStory<typeof Pack> = (args) => (
  <Pack {...args}>
    <Button variant="text">Button 1</Button>
    <Button variant="text">Button 1</Button>
    <Button variant="text">Button 1</Button>
  </Pack>
);

export const Basic = Template.bind({});

// eslint-disable-next-line import/no-default-export
export default story;
