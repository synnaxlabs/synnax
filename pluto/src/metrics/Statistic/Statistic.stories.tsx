// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { ComponentMeta, ComponentStory } from "@storybook/react";

import { Statistic } from ".";

const story: ComponentMeta<typeof Statistic> = {
  title: "Metrics/Statistic",
  component: Statistic,
  argTypes: {
    level: {
      control: {
        type: "select",
        options: ["h1", "h2", "h3", "h4", "h5", "p", "small"],
      },
    },
    variant: {
      control: {
        type: "select",
        options: ["primary", "error"],
      },
    },
  },
};

const Template: ComponentStory<typeof Statistic> = (props) => <Statistic {...props} />;

export const Primary = Template.bind({});
Primary.args = {
  value: 12,
  level: "h1",
  label: "Events",
  variant: "primary",
};

// eslint-disable-next-line import/no-default-export
export default story;
