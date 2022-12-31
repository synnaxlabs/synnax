// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { ComponentMeta, ComponentStory } from "@storybook/react";
import { AiOutlinePlus } from "react-icons/ai";
import { MdGrain } from "react-icons/md";

import { TypographyLevels } from "../Typography";

import { HeaderProps } from "./Header";

import { Header } from ".";

const story: ComponentMeta<typeof Header> = {
  title: "Atoms/Header",
  component: Header,
  argTypes: {
    level: {
      control: { type: "select" },
      options: TypographyLevels,
    },
    icon: {
      control: { type: "json" },
    },
  },
};

const Template: ComponentStory<typeof Header> = (args: HeaderProps) => (
  <Header {...args} />
);

export const Primary = Template.bind({});
Primary.args = {
  icon: <MdGrain />,
  children: "Heading",
  level: "p",
  divided: true,
  actions: [{ children: <AiOutlinePlus />, variant: "text" }],
};

export default story;
