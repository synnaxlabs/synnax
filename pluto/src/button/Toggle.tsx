// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/button/Toggle.css";

import { type ReactElement } from "react";

import { Button, type ButtonProps } from "@/button/Button";
import { CSS } from "@/css";
import { type Input } from "@/input";

export interface ToggleProps
  extends
    Input.Control<boolean, boolean>,
    Omit<ButtonProps, "value" | "onChange" | "variant"> {
  checkedVariant?: ButtonProps["variant"];
  uncheckedVariant?: ButtonProps["variant"];
  rightClickToggle?: boolean;
}

export const Toggle = ({
  value,
  onClick,
  onChange,
  checkedVariant = "outlined",
  uncheckedVariant,
  rightClickToggle = false,
  className,
  ...rest
}: ToggleProps): ReactElement => (
  <Button
    className={CSS(className, CSS.B("btn-toggle"), value && CSS.M("checked"))}
    onClick={(e) => {
      onClick?.(e);
      if (rightClickToggle) return;
      onChange(!value);
    }}
    onContextMenu={(e: React.MouseEvent) => {
      if (!rightClickToggle) return;
      e.preventDefault();
      e.stopPropagation();
      onChange(!value);
    }}
    variant={value ? checkedVariant : uncheckedVariant}
    {...rest}
  />
);
