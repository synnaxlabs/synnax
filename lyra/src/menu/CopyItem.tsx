// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Icon } from "@/icon";
import { nodeText, useMatches, usePathChildren } from "@/menu/filter";
import { useContext } from "@/menu/Menu";

/** Props for {@link CopyItem}. */
export interface CopyItemProps extends Button.CopyProps {
  /** The key given to the enclosing menu's `onChange`. */
  itemKey: string;
}

/**
 * A menu entry that copies text to the clipboard and swaps its icon to a check. It
 * takes its level, gap, and background from the enclosing {@link Menu}.
 */
export const CopyItem = ({
  className,
  itemKey,
  children,
  ...rest
}: CopyItemProps): ReactElement | null => {
  const { onClick: ctxOnClick, level = "p", gap, background } = useContext();
  const matches = useMatches(nodeText(children));
  // The copy icon leads, so the button takes no names of submenus for its icon.
  const content = usePathChildren(children, <Icon.Copy key="icon" />);
  const handleClick: Button.ButtonProps["onClick"] = () => ctxOnClick(itemKey);
  if (!matches) return null;
  return (
    <Button.Copy
      level={level}
      overflow="nowrap"
      variant="text"
      className={CSS.cls(CSS.B("menu-item"), className)}
      background={background}
      tooltip={null}
      gap={gap}
      propagateClick
      role="menuitem"
      {...rest}
      onClick={handleClick}
    >
      {content}
    </Button.Copy>
  );
};
