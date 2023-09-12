// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import type { Meta, StoryFn } from "@storybook/react";

import { Resize } from "@/resize";

const story: Meta<typeof Resize.Single> = {
  title: "Resize",
  component: Resize.Single,
};

const Template = (args: Resize.SingleProps): ReactElement => (
  <Resize.Single {...args} location="right">
    <h1>Resize</h1>
  </Resize.Single>
);

export const Primary: StoryFn<typeof Resize.Single> = Template.bind({});
Primary.args = {
  style: {
    position: "absolute",
    right: 0,
    height: "100%",
    minWidth: 200,
    maxWidth: 500,
  },
};

export const Multiple: StoryFn<typeof Resize.Multiple> = () => {
  const { props } = Resize.useMultiple({
    count: 3,
  });
  return (
    <Resize.Multiple
      {...props}
      style={{ border: "1px solid var(--pluto-white)", height: "100%" }}
    >
      <h1>Hello From One</h1>
      <h1>Hello From Two</h1>
      <h1>Hello From Three</h1>
    </Resize.Multiple>
  );
};

// eslint-disable-next-line import/no-default-export
export default story;
