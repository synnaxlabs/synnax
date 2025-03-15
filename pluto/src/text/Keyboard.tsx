// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/text/Keyboard.css";

import { type CSSProperties, type ReactElement } from "react";

import { CSS } from "@/css";
import { type text } from "@/text/core";
import { Text, type TextProps } from "@/text/Text";

export type KeyboardProps<L extends text.Level = "h1"> = TextProps<L>;

export const Keyboard = <L extends text.Level = "p">({
  className,
  level,
  children,
  style,
  ...rest
}: KeyboardProps<L>): ReactElement => {
  const iStyle: CSSProperties = {
    height: `calc(var(--pluto-${level}-size) * 1.7)`,
  };

  let rect = true;
  if (typeof children !== "string" || children.length === 1) {
    rect = false;
    iStyle.width = `calc(var(--pluto-${level}-size) * 1.7)`;
  }

  return (
    // @ts-expect-error - generic component errors
    <Text<L>
      className={CSS(className, CSS.BM("text", "keyboard"), rect && CSS.M("rect"))}
      level={level}
      style={{ ...style, ...iStyle }}
      {...rest}
    >
      {children}
    </Text>
  );
};
