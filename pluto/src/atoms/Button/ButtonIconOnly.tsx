// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { cloneElement, ReactElement } from "react";

import clsx from "clsx";

import { BaseButtonProps } from "./Button";

/** The props for the {@link ButtonIconOnly} */
export interface ButtonIconOnlyProps extends BaseButtonProps {
  children: ReactElement;
}

export const ButtonIconOnly = ({
  children,
  className,
  variant = "text",
  size = "medium",
  ...props
}: ButtonIconOnlyProps): JSX.Element => {
  return (
    <button
      className={clsx(
        "pluto-btn pluto-btn-icon",
        "pluto-btn--" + size,
        "pluto-btn--" + variant,
        className
      )}
      {...props}
    >
      {cloneElement(children, { className: "pluto-btn-icon__icon" })}
    </button>
  );
};
