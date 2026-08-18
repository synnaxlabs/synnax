// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/header/Header.css";

import { type text } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useMemo } from "react";

import { context } from "@/context";
import { CSS } from "@/css";
import { Flex } from "@/flex";

/** Props for {@link Header}. */
export interface HeaderProps extends Omit<Flex.BoxProps, "children" | "el"> {
  /** Type scale step for the title and its actions. Defaults to "h1". */
  level?: text.Level;
  /** Whether to draw a rule between the title and the actions. */
  divided?: boolean;
  bordered?: boolean;
  children: ReactNode | [ReactNode, ReactNode];
}

export interface ContextValue {
  divided: boolean;
  level: text.Level;
}

const [Context, useContext] = context.create<ContextValue>({
  defaultValue: { divided: false, level: "h1" },
  displayName: "Header.Context",
});
export { useContext };

/**
 * The bar at the top of a module. It gives its {@link Title} and {@link Actions} a
 * shared type scale, so their sizes stay in step.
 *
 * @example
 * <Header.Header level="h4">
 *   <Header.Title>Ranges</Header.Title>
 *   <Header.Actions><Button.Button onClick={add}><Icon.Add /></Button.Button></Header.Actions>
 * </Header.Header>
 * @param props.level - The font level for the header. See the {@link Typography.Text}
 * component for all possible levels. Default is "h1."
 * @param props.divided - If true, creates a divider between the start icon, header
 * text, and each action. Default is false.
 */
export const Header = ({
  className,
  level = "p",
  divided = false,
  bordered = true,
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
        className={CSS.cx(
          CSS.B("header"),
          bordered && CSS.bordered("bottom"),
          divided && CSS.BM("header", "divided"),
          className,
        )}
        {...rest}
      />
    </Context>
  );
};
