// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import clsx from "clsx";

import { StatusVariant } from "@/atoms/Status";
import { Text, TextProps } from "@/atoms/Typography";
import "./InputHelpText.css";

export interface InputHelpTextProps extends Omit<TextProps, "level"> {
  variant?: StatusVariant;
}

export const InputHelpText = ({
  className,
  variant = "error",
  ...props
}: InputHelpTextProps): JSX.Element => (
  <Text
    className={clsx(
      "pluto-input-help-text",
      `pluto-input-help-text--${variant}`,
      className
    )}
    level="small"
    {...props}
  />
);
