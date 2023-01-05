// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { Button } from "@/core/Button";
import { Space } from "@/core/Space";

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
