// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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
}: NavDrawerProps): JSX.Element => {
  const { direction } = useNavbar(props);
  const [activeKey, setActiveKey] = useState<string | undefined>(initialKey);
  const onClick = (key: string): void =>
    setActiveKey(key === activeKey ? undefined : key);
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
