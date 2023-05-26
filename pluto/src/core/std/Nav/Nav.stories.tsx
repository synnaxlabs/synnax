// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Meta, StoryFn } from "@storybook/react";
import { MdGrain } from "react-icons/md";

import { Nav } from ".";

import { Menu } from "@/core/std/Menu";

const story: Meta<typeof Nav.Bar> = {
  title: "Core/Nav",
  component: Nav.Bar,
};

export const LeftBar: StoryFn<typeof Nav.Bar> = () => (
  <Nav.Bar>
    <Nav.Bar.Start>
      <Menu>
        <Menu.Item.Icon itemKey="1">
          <MdGrain />
        </Menu.Item.Icon>
      </Menu>
    </Nav.Bar.Start>
    <Nav.Bar.Content>
      <Menu>
        <Menu.Item.Icon itemKey="1">
          <MdGrain />
        </Menu.Item.Icon>
      </Menu>
    </Nav.Bar.Content>
    <Nav.Bar.End>
      <Menu>
        <Menu.Item.Icon itemKey="1">
          <MdGrain />
        </Menu.Item.Icon>
      </Menu>
    </Nav.Bar.End>
  </Nav.Bar>
);

export const LeftDrawer: StoryFn<typeof Nav.Drawer> = () => {
  const props = Nav.useDrawer({
    initialKey: "2",
    items: [
      {
        key: "2",
        content: <h1>Hello</h1>,
        initialSize: 200,
      },
    ],
  });
  return (
    <Nav.Drawer location="bottom" {...props} style={{ position: "fixed", bottom: 0 }} />
  );
};

// eslint-disable-next-line import/no-default-export
export default story;
