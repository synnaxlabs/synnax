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

const story: Meta<typeof Align.Pack> = {
  title: "Pack",
  component: Align.Pack,
  argTypes: {
    direction: {
      control: { type: "select" },
      options: ["x", "y"],
    },
  },
};

export const Horizontal: StoryFn<typeof Align.Pack> = (args) => (
  <Align.Pack {...args} reverse>
    <Button.Button variant="text">Button 1</Button.Button>
    <Button.Button variant="text">Button 2</Button.Button>
    <Button.Button variant="text">Button 3</Button.Button>
  </Align.Pack>
);

export const Vertical: StoryFn<typeof Align.Pack> = (args) => (
  <Align.Pack {...args} direction="y">
    <Button.Button variant="text">Button 1</Button.Button>
    <Button.Button variant="text">Button 222</Button.Button>
    <Button.Button variant="text">Button 3</Button.Button>
  </Align.Pack>
);

export const Nested: StoryFn<typeof Align.Pack> = (args) => (
  <Align.Pack direction="y">
    <Align.Pack {...args}>
      <Button.Button variant="text">Button</Button.Button>
      <Button.Button variant="text">Button</Button.Button>
    </Align.Pack>
    <Align.Pack {...args}>
      <Button.Button variant="text">Button</Button.Button>
      <Button.Button variant="text">Button</Button.Button>
    </Align.Pack>
    <Align.Pack {...args}>
      <Button.Button variant="text">Button</Button.Button>
      <Button.Button variant="text">Button</Button.Button>
    </Align.Pack>
  </Align.Pack>
);

export const NestedX: StoryFn<typeof Align.Pack> = (args) => (
  <Align.Pack direction="x">
    <Align.Pack {...args} direction="y">
      <Button.Button variant="text">Button</Button.Button>
      <Button.Button variant="text">Button</Button.Button>
      <Button.Button variant="text">Button</Button.Button>
    </Align.Pack>
    <Align.Pack {...args} direction="y">
      <Button.Button variant="text">Button</Button.Button>
      <Button.Button variant="text">Button</Button.Button>
      <Button.Button variant="text">Button</Button.Button>
    </Align.Pack>
    <Align.Pack {...args} direction="y">
      <Button.Button variant="text">Button</Button.Button>
      <Button.Button variant="text">Button</Button.Button>
      <Button.Button variant="text">Button</Button.Button>
    </Align.Pack>
  </Align.Pack>
);

// eslint-disable-next-line import/no-default-export
export default story;
