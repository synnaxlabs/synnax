// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { useContext } from "@/header/Header";
import { Text } from "@/text";

export interface ButtonTitleProps
  extends Omit<Button.ButtonProps, "variant" | "size"> {}

/**
 * Header.Title.Use renders a clickable header title.
 *
 * @param props - The comonent props. The props for this component are identical
 * to {@link Button}, except the variant is always 'outlined' and that the component size
 * is automatically inferred from the 'level' prop passsed to the parent {@link Header}
 * component.
 */
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
      size={Text.LEVEL_COMPONENT_SIZES[level]}
      onClick={onClick}
      className={CSS(CSS.B("header-button-title"), className)}
      sharp
      {...rest}
    >
      {children}
    </Button.Button>
  );
};
