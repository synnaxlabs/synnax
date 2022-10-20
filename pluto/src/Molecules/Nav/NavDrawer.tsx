import clsx from "clsx";
import { ReactElement, useState } from "react";
import Button from "../../Atoms/Button/Button";
import Resize from "../../Atoms/Resize/Resize";
import Space from "../../Atoms/Space/Space";
import { swapDirection } from "../../util/spatial";
import NavBar, { NavBarProps, useNavBar } from "../Nav/NavBar";
import "./NavDrawer.css";

export type NavDrawerItem = {
  key: string;
  content: ReactElement;
  icon: ReactElement;
  minSize?: number;
  maxSize?: number;
  initialSize?: number;
};

export interface NavDrawerProps extends NavBarProps {
  items: NavDrawerItem[];
  initialKey?: string;
}

export default function NavDrawer({
  items = [],
  initialKey,
  children,
  ...props
}: NavDrawerProps) {
  const { direction } = useNavBar(props);
  const [activeKey, setActiveKey] = useState<string | undefined>(initialKey);
  const onClick = (key: string) =>
    setActiveKey(key === activeKey ? undefined : key);
  const activeItem = items.find((item) => item.key === activeKey);
  return (
    <NavBar.Context.Provider value={{ direction, location: props.location }}>
      <Space
        direction={swapDirection(direction)}
        empty
        reverse={props.location === "right" || props.location === "bottom"}
        className="pluto-navdrawer__container"
        align="stretch"
        style={{ height: "100%" }}
      >
        <NavBar {...props} withContext={false}>
          {children}
          <NavBar.Content className="pluto-navdrawer__menu">
            {items.map(({ key, icon }) => (
              <Button.IconOnly key={key} onClick={() => onClick(key)}>
                {icon}
              </Button.IconOnly>
            ))}
          </NavBar.Content>
        </NavBar>
        {activeItem && (
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
    </NavBar.Context.Provider>
  );
}
