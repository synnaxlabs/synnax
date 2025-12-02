// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/header/Header.css";

import { type ReactElement, type ReactNode, useMemo } from "react";

import { context } from "@/context";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { type Text } from "@/text";

export interface HeaderProps extends Omit<Flex.BoxProps, "children" | "el"> {
  level?: Text.Level;
  divided?: boolean;
  bordered?: boolean;
  padded?: boolean;
  children: ReactNode | [ReactNode, ReactNode];
}

export interface ContextValue {
  divided: boolean;
  level: Text.Level;
}

const [Context, useContext] = context.create<ContextValue>({
  defaultValue: { divided: false, level: "h1" },
  displayName: "Header.Context",
});
export { useContext };

/**
 * The container for a module header.
 *
 * @param props - The component props. All unused props will be passed down to the
 * {@link Space} containing the header.
 * @param props.level - The font level for the header. See the {@link Typography.Text}
 * component for all possible levels. Default is "h1."
 * @param props.divided - If true, creates a divider between the start icon, header text,
 * and each action. Default is false.
 */
export const Header = ({
  className,
  level = "p",
  divided = false,
  bordered = true,
  padded = false,
  ...rest
}: HeaderProps): ReactElement => {
  const value = useMemo(() => ({ level, divided }), [level, divided]);
  return (
    <Context value={value}>
      <Flex.Box
        el="header"
        x
        align="center"
        justify="between"
        className={CSS(
          CSS.B("header"),
          bordered && CSS.bordered("bottom"),
          divided && CSS.BM("header", "divided"),
          padded && CSS.BM("header", "padded"),
          className,
        )}
        {...rest}
      />
    </Context>
  );
};
