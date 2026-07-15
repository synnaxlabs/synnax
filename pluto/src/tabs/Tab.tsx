// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type KeyboardEventHandler,
  type MouseEventHandler,
  type ReactElement,
  useCallback,
} from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Menu } from "@/menu";
import { Select } from "@/select";
import { panelID, tabID, useFrameID } from "@/tabs/Frame";
import { useSelectorContext } from "@/tabs/Selector";
import { Triggers } from "@/triggers";

const PILL_BUTTON_PROPS = {
  variant: "outlined",
  rounded: true,
  contrast: 2,
  color: 1,
  textColor: 1,
} as const;

const DEFAULT_BUTTON_PROPS = {
  variant: "text",
  bordered: false,
  sharp: true,
} as const;

export interface TabProps extends Omit<Button.ButtonProps<"div">, "el" | "id"> {
  /** itemKey identifies the tab within its Frame's selection and content panels. */
  itemKey: string;
}

/**
 * Tab is a single selectable handle within a Selector. Clicking it (or pressing
 * Enter or Space while it is focused) selects it in the enclosing Frame. Children
 * define its contents: text, an icon, a {@link Close} button, or any
 * combination. When the tab heads an ordered multi-selection (its
 * key is first in the enclosing selection's array value), it is the focused tab
 * and colors itself with the primary theme color.
 */
export const Tab = ({
  itemKey,
  className,
  children,
  onClick,
  onKeyDown,
  ...rest
}: TabProps): ReactElement => {
  const frameID = useFrameID("Tabs.Tab");
  const { size, variant } = useSelectorContext("Tabs.Tab");
  const { selected, focused, onSelect } = Select.useItemState(itemKey);
  const handleClick = useCallback<MouseEventHandler<HTMLDivElement>>(
    (e) => {
      onClick?.(e);
      onSelect();
    },
    [onClick, onSelect],
  );
  const handleKeyDown = useCallback<KeyboardEventHandler<HTMLDivElement>>(
    (e) => {
      onKeyDown?.(e);
      if (e.target !== e.currentTarget || e.defaultPrevented) return;
      const key = Triggers.eventKey(e);
      if (key !== "Enter" && key !== "Space") return;
      e.preventDefault();
      onSelect();
    },
    [onKeyDown, onSelect],
  );
  const isPill = variant === "pill";
  const variantProps = isPill ? PILL_BUTTON_PROPS : DEFAULT_BUTTON_PROPS;
  return (
    <Button.Button
      el="div"
      id={tabID(frameID, itemKey)}
      role="tab"
      aria-selected={selected}
      aria-controls={panelID(frameID, itemKey)}
      data-tab-key={itemKey}
      data-menu-key={itemKey}
      tabIndex={selected ? 0 : -1}
      size={size}
      className={CSS(
        CSS.BE("tabs", "tab"),
        Menu.CONTEXT_TARGET,
        selected && Menu.CONTEXT_SELECTED,
        CSS.selected(selected),
        CSS.altColor(focused),
        className,
      )}
      justify="center"
      align="center"
      empty
      preventClick={selected}
      onClick={handleClick}
      onKeyDown={handleKeyDown}
      {...variantProps}
      borderColor={isPill ? (selected ? 7 : 5) : undefined}
      {...rest}
    >
      {children}
    </Button.Button>
  );
};
