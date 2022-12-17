import { ReactElement, useState } from "react";

import clsx from "clsx";

import { Button, Resize, Space } from "../../atoms";
import { swapDirection } from "../../util/spatial";

import { Navbar, NavbarProps, useNavbar } from "./Navbar";
import "./Navdrawer.css";

export interface NavDrawerItem {
  key: string;
  content: ReactElement;
  icon: ReactElement;
  minSize?: number;
  maxSize?: number;
  initialSize?: number;
}

export interface NavDrawerProps extends NavbarProps {
  items: NavDrawerItem[];
  initialKey?: string;
}

export const NavDrawer = ({
  items = [],
  initialKey,
  children,
  ...props
}: NavDrawerProps) => {
  const { direction } = useNavbar(props);
  const [activeKey, setActiveKey] = useState<string | undefined>(initialKey);
  const onClick = (key: string) => setActiveKey(key === activeKey ? undefined : key);
  const activeItem = items.find((item) => item.key === activeKey);
  return (
    <Navbar.Context.Provider value={{ direction, location: props.location }}>
      <Space
        direction={swapDirection(direction)}
        empty
        reverse={props.location === "right" || props.location === "bottom"}
        className="pluto-navdrawer__container"
        align="stretch"
        style={{ height: "100%" }}
      >
        <Navbar {...props} withContext={false}>
          {children}
          <Navbar.Content className="pluto-navdrawer__menu">
            {items.map(({ key, icon }) => (
              <Button.IconOnly key={key} onClick={() => onClick(key)}>
                {icon}
              </Button.IconOnly>
            ))}
          </Navbar.Content>
        </Navbar>
        {activeItem != null && (
          <Resize
            className={clsx(
              "pluto-navdrawer__content",
              `pluto-navdrawer__content--${direction}`,
              `pluto-navdrawer__content--${props.location}`
            )}
            location={props.location}
            {...activeItem}
          >
            {activeItem.content}
          </Resize>
        )}
      </Space>
    </Navbar.Context.Provider>
  );
};
