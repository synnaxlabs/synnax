import { ReactElement, useState } from "react";

import clsx from "clsx";

import { NavbarProps, useNavbar } from "./Navbar";
import { NavMenuItem } from "./NavMenu";

import "./Navdrawer.css";

import { Resize } from "@/atoms";
import { ResizePanelProps } from "@/atoms/Resize/Resize";

export interface NavDrawerContent {
  key: string;
  content: ReactElement;
  minSize?: number;
  maxSize?: number;
  initialSize?: number;
}

export interface NavDrawerItem extends NavDrawerContent, NavMenuItem {}

export interface UseNavDrawerProps {
  initialKey?: string;
  items: NavDrawerItem[];
}

export interface UseNavDrawerReturn {
  activeItem?: NavDrawerContent;
  menuItems?: NavMenuItem[];
  onSelect?: (key: string) => void;
}

export interface NavDrawerProps
  extends Omit<NavbarProps, "onSelect">,
    UseNavDrawerReturn,
    Partial<Pick<ResizePanelProps, "onResize">> {}

export const useNavDrawer = ({
  items,
  initialKey,
}: UseNavDrawerProps): UseNavDrawerReturn => {
  const [activeKey, setActiveKey] = useState<string | undefined>(initialKey);
  const handleSelect = (key: string): void =>
    setActiveKey(key === activeKey ? undefined : key);
  const activeItem = items.find((item) => item.key === activeKey);
  return { onSelect: handleSelect, activeItem, menuItems: items };
};

export const Navdrawer = ({
  activeItem,
  menuItems = [],
  children,
  onSelect,
  onResize,
  ...props
}: NavDrawerProps): JSX.Element | null => {
  const { direction } = useNavbar(props);
  if (activeItem == null) return null;
  const { content, maxSize, minSize, initialSize } = activeItem;
  return (
    <Resize
      className={clsx(
        "pluto-navdrawer__content",
        `pluto-navdrawer__content--${direction}`,
        `pluto-navdrawer__content--${props.location}`
      )}
      onResize={onResize}
      minSize={minSize}
      maxSize={maxSize}
      initialSize={initialSize}
      {...props}
    >
      {content}
    </Resize>
  );
};
