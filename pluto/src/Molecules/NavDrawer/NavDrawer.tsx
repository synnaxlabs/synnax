import clsx from "clsx";
import { ReactElement, useState } from "react";
import Button from "../../Atoms/Button/Button";
import ResizePanel from "../../Atoms/ResizePanel/ResizePanel";
import Space from "../../Atoms/Space/Space";
import Navbar, { NavbarProps, useNavbar } from "../Navbar/Navbar";
import "./Navdrawer.css";

type NavdrawerItem = {
  key: string;
  content: ReactElement;
  icon: ReactElement;
  minWidth?: number;
  maxWidth?: number;
  initialWidth?: number;
};

export interface NavdrawerProps extends NavbarProps {
  items: NavdrawerItem[];
  initialKey?: string;
}

export default function Navdrawer({
  items = [],
  initialKey,
  children,
  ...props
}: NavdrawerProps) {
  const { direction } = useNavbar(props);
  const [activeKey, setActiveKey] = useState<string | undefined>(initialKey);
  const onClick = (key: string) =>
    setActiveKey(key === activeKey ? undefined : key);
  const activeItem = items.find((item) => item.key === activeKey);
  return (
    <Navbar.Context.Provider value={{ direction, location: props.location }}>
      <Space direction="horizontal" empty reverse={props.location === "right"}>
        <Navbar {...props} context={false}>
          {children}
          <Navbar.Content className="pluto-navdrawer__menu">
            {items.map(({ key, icon }) => (
              <Button.IconOnly key={key} onClick={() => onClick(key)}>
                {icon}
              </Button.IconOnly>
            ))}
          </Navbar.Content>
        </Navbar>
        {activeItem && (
          <ResizePanel
            className={clsx(
              "pluto-navdrawer__content",
              `pluto-navdrawer__content--${direction}`,
              `pluto-navdrawer__content--${props.location}`
            )}
            location={props.location}
            {...activeItem}
          >
            {activeItem.content}
          </ResizePanel>
        )}
      </Space>
    </Navbar.Context.Provider>
  );
}
