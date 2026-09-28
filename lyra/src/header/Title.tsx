// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type text } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { useContext } from "@/header/Header";
import { Text } from "@/text";

/** Props for {@link Title}. */
export interface TitleProps extends Omit<Text.TextProps, "divided" | "level"> {
  /** Overrides the level the enclosing {@link Header} sets. */
  level?: text.Level;
}

/** The title of a {@link Header}. It takes its type scale step from the header. */
export const Title = ({
  className,
  level: propsLevel,
  ...rest
}: TitleProps): ReactElement => {
  const { level } = useContext();
  return (
    <Text.Text
      className={CSS.cls(CSS.BE("header", "text"), className)}
      level={propsLevel ?? level}
      gap={1.5}
      {...rest}
    />
  );
};
