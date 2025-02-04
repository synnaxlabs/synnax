// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/components/Toolbar/Toolbar.css";

import { Header, type Icon } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/css";

export const ToolbarHeader = (
  props: Omit<Header.HeaderProps, "level" | "divided">,
): ReactElement => (
  <Header.Header
    className={CSS.B("toolbar-header")}
    level="h5"
    shrink={false}
    {...props}
  />
);

export interface ToolbarTitleProps extends Pick<Header.TitleProps, "children"> {
  icon: Icon.Element;
}

export const ToolbarTitle = ({ icon, children }: ToolbarTitleProps): ReactElement => (
  <Header.Title shade={8} weight={500} startIcon={icon}>
    {children}
  </Header.Title>
);
