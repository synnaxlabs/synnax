// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/button/CreateListItem.css";

import { type ReactElement } from "react";

import { Button, type ButtonProps } from "@/button/Button";
import { CSS } from "@/css";
import { Icon } from "@/icon";

/** Props for {@link CreateListItem}. */
export interface CreateListItemProps extends Omit<ButtonProps, "size"> {
  /** The scale of the list the button sits under. A small item is a small button. */
  size?: "small" | "medium" | "large";
}

// A large item keeps medium text in its larger box.
const BUTTON_SIZES: Record<
  NonNullable<CreateListItemProps["size"]>,
  ButtonProps["size"]
> = { small: "small", medium: "medium", large: "medium" };

/** A pinned create action styled to read as a list's last item. */
export const CreateListItem = ({
  className,
  size = "medium",
  children,
  ...rest
}: CreateListItemProps): ReactElement => (
  <Button
    size={BUTTON_SIZES[size]}
    variant="text"
    textColor={9}
    justify="start"
    className={CSS.cls(
      CSS.B("create-list-item"),
      CSS.BM("create-list-item", size),
      className,
    )}
    {...rest}
  >
    <Icon.Add />
    {children}
  </Button>
);
