// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { forwardRef, ReactElement } from "react";

import clsx from "clsx";

import { CSS } from "@/core/css";
import type { BaseButtonProps } from "@/core/std/Button/Button";

/** The props for the {@link ButtonIcon} */
export interface ButtonIconProps extends BaseButtonProps {
  children: ReactElement;
}

export const ButtonIcon = forwardRef<HTMLButtonElement, ButtonIconProps>(
  (
    { children, className, variant = "text", size = "medium", sharp = false, ...props },
    ref
  ): ReactElement => (
    <button
      ref={ref}
      className={clsx(
        CSS.B("btn"),
        CSS.B("btn-icon"),
        CSS.size(size),
        CSS.sharp(sharp),
        CSS.BM("btn", variant),
        className
      )}
      {...props}
    >
      {children}
    </button>
  )
);
ButtonIcon.displayName = "ButtonIcon";
