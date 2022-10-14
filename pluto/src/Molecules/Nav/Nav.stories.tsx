import { ComponentMeta, ComponentStory } from "@storybook/react";
import { MdGrain } from "react-icons/md";
import { Nav, NavDrawerProps } from ".";
import { Header } from "../../Atoms";
import { NavBarProps } from "./NavBar";

export default {
  title: "Molecules/Nav",
  component: Nav.Bar,
} as ComponentMeta<typeof Nav.Bar>;

const Template = (args: NavBarProps) => <Nav.Bar {...args} />;

export const LeftBar: ComponentStory<typeof Nav.Bar> = Template.bind({});

export const LeftDrawer: ComponentStory<typeof Nav.Drawer> = (
  args: NavDrawerProps
) => {
  return (
    <Nav.Drawer
      initialKey="2"
      location="bottom"
      items={[
        {
          key: "2",
          icon: <MdGrain />,
          content: (
            <Header
              icon={<MdGrain />}
              level="p"
              style={{ color: "white" }}
              divided
            >
              Hello
            </Header>
          ),
        },
      ]}
    ></Nav.Drawer>
  );
};
