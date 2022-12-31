// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { ComponentMeta, ComponentStory } from "@storybook/react";

import { ResizePanelProps } from "./Resize";

import { Resize } from ".";

const story: ComponentMeta<typeof Resize> = {
  title: "Atoms/Resize",
  component: Resize,
};

const Template = (args: ResizePanelProps): JSX.Element => (
  <Resize {...args}>
    <h1>Resize</h1>
  </Resize>
);

export const Primary: ComponentStory<typeof Resize> = Template.bind({});
Primary.args = {
  style: {
    height: "100%",
  },
};

export const Multiple: ComponentStory<typeof Resize.Multiple> = () => {
  const { props } = Resize.useMultiple({ initialSizes: [100, 200], count: 3 });
  return (
    <Resize.Multiple
      {...props}
      style={{ border: "1px solid var(--pluto-gray-m2)", height: "100%" }}
    >
      <h1>Hello From One</h1>
      <h1>Hello From Two</h1>
      <h1>Hello From Three</h1>
    </Resize.Multiple>
  );
};

export default story;
