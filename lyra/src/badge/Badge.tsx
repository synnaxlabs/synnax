// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/badge/Badge.css";

import { type ComponentPropsWithoutRef, type ReactElement } from "react";

import { CSS } from "@/css";
import { type Variant } from "@/status/status";

/** Props for the {@link Badge} component. */
export interface BadgeProps extends ComponentPropsWithoutRef<"span"> {
  /** tints the badge with the color of a status variant. Without one, it is gray. */
  status?: Variant;
}

/**
 * Shows a short fact that the program writes about the item beside it, such as
 * "Connected" or "Auto". It is not interactive.
 */
export const Badge = ({ status, className, ...rest }: BadgeProps): ReactElement => (
  <span
    className={CSS.cls(
      CSS.B("badge"),
      status != null && CSS.M("status", status),
      className,
    )}
    {...rest}
  />
);
