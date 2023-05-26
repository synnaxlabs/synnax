// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createContext, ReactElement, ReactNode, useContext } from "react";

import { CSS } from "@/core/css";
import { Space, SpaceProps } from "@/core/std/Space";
import { TypographyLevel } from "@/core/std/Typography";

export interface HeaderProps extends Omit<SpaceProps, "children" | "el"> {
  level?: TypographyLevel;
  divided?: boolean;
  children: ReactNode | [ReactNode, ReactNode];
}

export interface HeaderContextValue {
  divided: boolean;
  level: TypographyLevel;
}

const HeaderContext = createContext<HeaderContextValue>({
  divided: false,
  level: "h1",
});

export const useHeaderContext = (): HeaderContextValue => useContext(HeaderContext);

export const Header = ({
  children,
  className,
  level = "h1",
  divided = false,
  ...props
}: HeaderProps): ReactElement => (
  <HeaderContext.Provider value={{ level, divided }}>
    <Space
      el="header"
      direction="x"
      justify="spaceBetween"
      className={CSS(
        CSS.B("header"),
        CSS.bordered("bottom"),
        divided && CSS.BM("header", "divided"),
        className
      )}
      {...props}
    >
      {children}
    </Space>
  </HeaderContext.Provider>
);
