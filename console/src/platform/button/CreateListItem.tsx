// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/button/CreateListItem.css";

import { Button, Icon } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";

export interface CreateListItemProps extends Omit<Button.ButtonProps, "size"> {
  /** Item scale matching the list the button sits under. */
  size?: "small" | "medium" | "large";
}

/** A pinned create action styled to read as a list's last item. */
export const CreateListItem = ({
  className,
  size = "medium",
  children,
  ...rest
}: CreateListItemProps): ReactElement => (
  <Button.Button
    variant="text"
    textColor={9}
    justify="start"
    className={CSS.cls(CSS.B("create-list-item"), CSS.M(size), className)}
    {...rest}
  >
    <Icon.Add />
    {children}
  </Button.Button>
);
