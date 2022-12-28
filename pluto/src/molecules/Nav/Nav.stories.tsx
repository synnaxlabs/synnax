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

export default story;
