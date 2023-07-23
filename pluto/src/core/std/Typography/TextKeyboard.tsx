// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { CSS } from "@/core/css";
import { TextProps, Text } from "@/core/std/Typography/Text";
import { TypographyLevel } from "@/core/std/Typography/types";

import "@/core/std/Typography/TextKeyboard.css";

export type TextKeyboardProps<L extends TypographyLevel = "h1"> = TextProps<L>;

export const TextKeyboard = <L extends TypographyLevel = "p">({
  className,
  level = "p" as L,
  ...props
}: TextKeyboardProps<L>): ReactElement => {
  return (
    <>
      {/** @ts-expect-error **/}
      <Text<L>
        level={level}
        className={CSS(className, CSS.BM("text", "keyboard"))}
        {...props}
      />
    </>
  );
};
