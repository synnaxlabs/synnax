// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/text/Keyboard.css";

import { type ReactElement } from "react";

import { CSS } from "@/css";
import { type text } from "@/text/core";
import { Text, type TextProps } from "@/text/Text";

export type KeyboardProps<L extends text.Level = "h1"> = TextProps<L>;

export const Keyboard = <L extends text.Level = "p">({
  className,
  ...props
}: KeyboardProps<L>): ReactElement => (
  // @ts-expect-error - generic component errors
  <Text<L> className={CSS(className, CSS.BM("text", "keyboard"))} {...props} />
);
