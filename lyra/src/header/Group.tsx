// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/header/Group.css";

import { type ReactElement, type ReactNode } from "react";

import { Button } from "@/button";
import { Caret } from "@/caret";
import { CSS } from "@/css";
import { Text } from "@/text";
import { isInputOrContentEditable } from "@/util";

/** Props for {@link Group}. */
export interface GroupProps extends Omit<
  Button.ButtonProps<"div">,
  "el" | "variant" | "onClick" | "children"
> {
  /** Whether the group under the heading is open. */
  open: boolean;
  /** Called with the next open state when the user clicks the heading. */
  onOpenChange: (open: boolean) => void;
  /** Buttons before the caret. A click on one does not open or close the group. */
  actions?: ReactNode;
  /** The title, then an optional count and badge. A plain string or span title takes
   * the weight of a button label. A click inside a {@link Text.Editable} title while
   * it is editing does not open or close the group. */
  children: ReactNode;
}

/**
 * The heading of a group that opens and closes, such as one palette in a list of
 * colors. A click anywhere on it, except on its actions, opens or closes the group.
 *
 * @example
 * <Header.Group open={open} onOpenChange={setOpen} actions={<Button.Button />}>
 *   Plant
 *   <Text.Text level="tiny" color={9}>4</Text.Text>
 * </Header.Group>
 */
export const Group = ({
  open,
  onOpenChange,
  actions,
  className,
  children,
  level = "p",
  ...rest
}: GroupProps): ReactElement => (
  <Text.Text
    el="div"
    level={level}
    x
    align="center"
    gap="small"
    className={CSS.cls(CSS.B("header-group"), className)}
  >
    <Button.Button
      el="div"
      role="button"
      tabIndex={0}
      variant="text"
      justify="start"
      gap="small"
      level={level}
      aria-expanded={open}
      onClick={(e) => {
        if (!isInputOrContentEditable(e)) onOpenChange(!open);
      }}
      className={CSS.BE("header-group", "toggle")}
      {...rest}
    >
      {children}
    </Button.Button>
    {actions != null && (
      <Text.Text
        el="div"
        level={level}
        x
        align="center"
        gap="tiny"
        className={CSS.BE("header-group", "actions")}
      >
        {actions}
      </Text.Text>
    )}
    <Caret.Animated
      enabled={open}
      enabledLoc="bottom"
      disabledLoc="left"
      className={CSS.BE("header-group", "caret")}
    />
  </Text.Text>
);
