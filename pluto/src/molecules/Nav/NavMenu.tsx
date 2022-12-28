import { ReactElement } from "react";

import { Button, Space } from "@/atoms";

import "./NavMenu.css";

export interface NavMenuItem {
  key: string;
  icon: ReactElement;
}

export interface NavMenuProps {
  items: NavMenuItem[];
  onSelect?: (key: string) => void;
}

export const NavMenu = ({ items, onSelect }: NavMenuProps): JSX.Element => (
  <Space className="pluto-nav-menu">
    {items.map(({ key, icon }) => (
      <Button.IconOnly key={key} onClick={() => onSelect?.(key)}>
        {icon}
      </Button.IconOnly>
    ))}
  </Space>
);
