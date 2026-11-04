// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/Auto.css";

import { type ComponentPropsWithoutRef, type ReactElement } from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Icon } from "@/icon";

/** Props for the {@link AutoChip} component. */
export interface AutoChipProps extends Omit<
  ComponentPropsWithoutRef<"span">,
  "children"
> {}

/**
 * Marks a field whose value the program picks until a person sets one. Place it where a
 * unit sits, such as an input's `endContent`.
 */
export const AutoChip = ({ className, ...rest }: AutoChipProps): ReactElement => (
  <span className={CSS.cls(CSS.B("input-auto-chip"), className)} {...rest}>
    Auto
  </span>
);

/** Props for the {@link AutoButton} component. */
export interface AutoButtonProps extends Omit<
  Button.ButtonProps,
  "children" | "variant" | "tooltip"
> {}

/** Puts a field back in auto. Place it beside a field whose value a person set. */
export const AutoButton = ({ className, ...rest }: AutoButtonProps): ReactElement => (
  <Button.Button
    variant="outlined"
    tooltip="Auto"
    className={CSS.cls(CSS.B("input-auto-button"), className)}
    {...rest}
  >
    <Icon.Auto />
  </Button.Button>
);
