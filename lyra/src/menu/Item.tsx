// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/menu/Item.css";

import { type ReactElement } from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { useContext } from "@/menu/Menu";

export interface ItemProps extends Button.ButtonProps {
  itemKey: string;
}

export const Item = ({
  itemKey,
  className,
  onClick,
  size,
  ...rest
}: ItemProps): ReactElement => {
  const { onClick: ctxOnClick, selected, level = "p", gap, background } = useContext();
  const handleClick: Button.ButtonProps["onClick"] = (e) => {
    ctxOnClick(itemKey);
    onClick?.(e);
  };
  const _selected = selected === itemKey;
  return (
    <Button.Button
      contrast={background}
      level={level}
      overflow="nowrap"
      onClick={handleClick}
      variant="text"
      className={CSS(CSS.B("menu-item"), CSS.selected(_selected), className)}
      size={size}
      gap={gap}
      propagateClick
      {...rest}
    />
  );
};
