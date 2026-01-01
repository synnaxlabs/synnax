// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { CSS } from "@/css";
import { useContext } from "@/header/Header";
import { Text } from "@/text";

export interface TitleProps extends Omit<Text.TextProps, "divided" | "level"> {
  level?: Text.Level;
}

/**
 * Renders the title for the header component.
 *
 * @param props - The component props. The props for this component are identical
 * to the {@link Typography.TextWithIcon} component, except that the 'level', and
 * 'divider' props are inherited from the parent {@link Header} component.
 */
export const Title = ({
  className,
  level: propsLevel,
  ...rest
}: TitleProps): ReactElement => {
  const { level } = useContext();
  return (
    <Text.Text
      className={CSS(CSS.BE("header", "text"), className)}
      level={propsLevel ?? level}
      gap={1.5}
      {...rest}
    />
  );
};
