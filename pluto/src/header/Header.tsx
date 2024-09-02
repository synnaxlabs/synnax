// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/header/Header.css";

import {
  createContext,
  type ReactElement,
  type ReactNode,
  useContext as reactUseContext,
} from "react";

import { Align } from "@/align";
import { CSS } from "@/css";
import { type Text } from "@/text";

export interface HeaderProps extends Omit<Align.SpaceProps, "children" | "el"> {
  level?: Text.Level;
  divided?: boolean;
  children: ReactNode | [ReactNode, ReactNode];
}

export interface ContextValue {
  divided: boolean;
  level: Text.Level;
}

const Context = createContext<ContextValue>({
  divided: false,
  level: "h1",
});

export const useContext = (): ContextValue => reactUseContext(Context);

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
  children,
  className,
  level = "h1",
  divided = false,
  ...props
}: HeaderProps): ReactElement => (
  <Context.Provider value={{ level, divided }}>
    <Align.Space
      el="header"
      direction="x"
      justify="spaceBetween"
      className={CSS(
        CSS.B("header"),
        CSS.bordered("bottom"),
        divided && CSS.BM("header", "divided"),
        className,
      )}
      {...props}
    >
      {children}
    </Align.Space>
  </Context.Provider>
);
