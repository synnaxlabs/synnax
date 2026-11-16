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
import { nodeText, useMatches } from "@/menu/filter";
import { useContext } from "@/menu/Menu";
import { useSubmenusContext } from "@/menu/submenus";

export interface ItemProps extends Button.ButtonProps {
  itemKey: string;
  /**
   * The item's name as plain text, for children that hold no text. The filter of a
   * menu matches it. Defaults to the text in the children.
   */
  textValue?: string;
}

export const Item = ({
  itemKey,
  textValue,
  className,
  onClick,
  onPointerMove,
  onPointerLeave,
  size,
  ...rest
}: ItemProps): ReactElement | null => {
  const { onClick: ctxOnClick, selected, level = "p", gap } = useContext();
  const { aiming, item } = useSubmenusContext();
  const matches = useMatches(textValue ?? nodeText(rest.children));
  const handlers = item(itemKey);
  const handleClick: Button.ButtonProps["onClick"] = (e) => {
    ctxOnClick(itemKey);
    onClick?.(e);
  };
  const _selected = selected === itemKey;
  if (!matches) return null;
  return (
    <Button.Button
      level={level}
      overflow="nowrap"
      onClick={handleClick}
      variant="text"
      className={CSS.cls(
        CSS.B("menu-item"),
        CSS.selected(_selected),
        aiming && CSS.BM("menu-item", "aiming"),
        className,
      )}
      size={size}
      gap={gap}
      propagateClick
      role="menuitem"
      onPointerMove={(e) => {
        onPointerMove?.(e);
        handlers.onPointerMove(e);
      }}
      onPointerLeave={(e) => {
        onPointerLeave?.(e);
        handlers.onPointerLeave(e);
      }}
      {...rest}
    />
  );
};
