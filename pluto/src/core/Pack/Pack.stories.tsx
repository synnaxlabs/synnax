// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Meta, StoryFn } from "@storybook/react";

import { Pack } from ".";

import { Button } from "@/core/Button";

const story: Meta<typeof Pack> = {
  title: "Core/Pack",
  component: Pack,
  argTypes: {
    direction: {
      control: { type: "select" },
      options: ["x", "y"],
    },
  },
};

export const Horizontal: StoryFn<typeof Pack> = (args) => (
  <Pack {...args} reverse>
    <Button variant="text">Button 1</Button>
    <Button variant="text">Button 2</Button>
    <Button variant="text">Button 3</Button>
  </Pack>
);

export const Vertical: StoryFn<typeof Pack> = (args) => (
  <Pack {...args} direction="y">
    <Button variant="text">Button 1</Button>
    <Button variant="text">Button 2</Button>
    <Button variant="text">Button 3</Button>
  </Pack>
);

export const Nested: StoryFn<typeof Pack> = (args) => (
  <Pack direction="y">
    <Pack {...args}>
      <Button variant="text">Button</Button>
      <Button variant="text">Button</Button>
    </Pack>
    <Pack>
      <Button variant="text">Button</Button>
      <Button variant="text">Button</Button>
    </Pack>
    <Pack>
      <Button variant="text">Button</Button>
      <Button variant="text">Button</Button>
    </Pack>
  </Pack>
);

// eslint-disable-next-line import/no-default-export
export default story;
