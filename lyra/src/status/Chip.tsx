// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/status/Chip.css";

import { type ComponentPropsWithoutRef, type ReactElement } from "react";

import { CSS } from "@/css";
import { type Variant } from "@/status/status";

/** Props for the {@link Chip} component. */
export interface ChipProps extends ComponentPropsWithoutRef<"span"> {
  variant: Variant;
}

/** A short word, such as "Connected", tinted with the color of a status variant. */
export const Chip = ({ variant, className, ...rest }: ChipProps): ReactElement => (
  <span
    className={CSS.cls(CSS.B("status-chip"), CSS.M("status", variant), className)}
    {...rest}
  />
);
