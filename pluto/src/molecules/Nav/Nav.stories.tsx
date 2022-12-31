// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { ComponentMeta, ComponentStory } from "@storybook/react";
import { MdGrain } from "react-icons/md";

import { Header } from "../../atoms";

import { NavbarProps } from "./Navbar";

import { Nav } from ".";

const story: ComponentMeta<typeof Nav.Bar> = {
  title: "Molecules/Nav",
  component: Nav.Bar,
};

const Template = (args: NavbarProps): JSX.Element => <Nav.Bar {...args} />;

export const LeftBar: ComponentStory<typeof Nav.Bar> = Template.bind({});

export const LeftDrawer: ComponentStory<typeof Nav.Drawer> = () => {
  return (
    <Nav.Drawer
      initialKey="2"
      location="bottom"
      items={[
        {
          key: "2",
          icon: <MdGrain />,
          content: (
            <Header icon={<MdGrain />} level="p" style={{ color: "white" }} divided>
              Hello
            </Header>
          ),
        },
      ]}
    ></Nav.Drawer>
  );
};

export default story;
