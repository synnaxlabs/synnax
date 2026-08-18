// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Button } from "@/button";
import { TEXT_LEVEL_SIZES } from "@/component/text";
import { CSS } from "@/css";
import { useContext } from "@/header/Header";

/** Props for {@link ButtonTitle}. */
export interface ButtonTitleProps extends Omit<
  Button.ButtonProps,
  "variant" | "size"
> {}

/** A {@link Title} the user can click. It takes its size from the enclosing header. */
export const ButtonTitle = ({
  children = "",
  className,
  onClick,
  ...rest
}: ButtonTitleProps): ReactElement => {
  const { level } = useContext();
  return (
    <Button.Button
      variant="text"
      size={TEXT_LEVEL_SIZES[level]}
      onClick={onClick}
      className={CSS(CSS.B("header-button-title"), className)}
      sharp
      {...rest}
    >
      {children}
    </Button.Button>
  );
};
