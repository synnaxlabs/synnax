// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { ComponentMeta, ComponentStory } from "@storybook/react";
import { MdGrain } from "react-icons/md";

import { NavbarProps } from "./Navbar";

import { Nav } from ".";

const story: ComponentMeta<typeof Nav.Bar> = {
  title: "Core/Nav",
  component: Nav.Bar,
};

const Template = (args: NavbarProps): JSX.Element => <Nav.Bar {...args} />;

export const LeftBar: ComponentStory<typeof Nav.Bar> = Template.bind({
  children: (
    <Nav.Bar.Content>
      <Nav.Menu
        items={[
          {
            key: "1",
            icon: <MdGrain />,
          },
        ]}
      />
    </Nav.Bar.Content>
  ),
});

export const LeftDrawer: ComponentStory<typeof Nav.Drawer> = () => {
  const props = Nav.useDrawer({
    initialKey: "2",
    items: [
      {
        key: "2",
        icon: <MdGrain />,
        content: <h1>Helllo</h1>,
      },
    ],
  });
  return <Nav.Drawer location="bottom" {...props} />;
};

// eslint-disable-next-line import/no-default-export
export default story;
