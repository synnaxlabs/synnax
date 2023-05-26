// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import "@/core/Input/InputHelpText.css";
import { StatusVariant } from "@/core/Status";
import { Text, TextProps } from "@/core/Typography";
import { CSS } from "@/css";

export interface InputHelpTextProps extends Omit<TextProps<"small">, "level" | "ref"> {
  variant?: StatusVariant;
}

export const InputHelpText = ({
  className,
  variant = "error",
  ...props
}: InputHelpTextProps): ReactElement => (
  <Text<"small">
    className={CSS(
      CSS.B("input-help-text"),
      CSS.BM("input-help-text", variant),
      className
    )}
    level="small"
    {...props}
  />
);
