// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { ComponentMeta, ComponentStory } from "@storybook/react";

import { HexagonBar } from "./Hexagon";

const story: ComponentMeta<typeof HexagonBar> = {
  title: "Metrics/Hexagon",
  component: HexagonBar,
};

const Template: ComponentStory<typeof HexagonBar> = (args) => <HexagonBar {...args} />;

export const Primary = Template.bind({});
Primary.args = {
  strokeWidth: 5,
  width: "50%",
  metrics: [
    {
      name: "Memory",
      value: 10,
      max: 100,
      units: "GB",
    },
    {
      name: "CPU",
      value: 30,
      max: 100,
      units: "%",
    },
    {
      name: "CPU",
      value: 65,
      max: 100,
      units: "%",
    },
  ],
};

export default story;
