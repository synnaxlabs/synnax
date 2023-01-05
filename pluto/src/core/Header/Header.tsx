// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Children, createContext, ReactNode, useContext } from "react";

import clsx from "clsx";

import { Space, SpaceProps } from "@/core/Space";
import { TypographyLevel } from "@/core/Typography";

export interface HeaderProps extends Omit<SpaceProps, "children"> {
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
}: HeaderProps): JSX.Element => {
  const _children = Children.toArray(children) as JSX.Element[];
  return (
    <HeaderContext.Provider value={{ level, divided }}>
      <Space
        direction="horizontal"
        justify="spaceBetween"
        className={clsx(`pluto-header pluto-bordered--bottom`, className)}
        {...props}
      >
        {_children[0]}
        {_children.length > 1 && _children[1]}
      </Space>
    </HeaderContext.Provider>
  );
};
