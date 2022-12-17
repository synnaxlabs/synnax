import type { ComponentMeta, ComponentStory } from "@storybook/react";
import { MdGrain } from "react-icons/md";

import { Header } from "../../atoms";

import { NavbarProps } from "./NavBar";

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
