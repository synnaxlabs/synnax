// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Header } from "@synnaxlabs/pluto";

export const ToolbarHeader = (
  props: Omit<Header.HeaderProps, "level" | "divided">
): ReactElement => <Header.Header level="h4" {...props} />;

export interface ToolbarTitleProps extends Pick<Header.TitleProps, "children"> {
  icon: ReactElement;
}

export const ToolbarTitle = ({ icon, children }: ToolbarTitleProps): ReactElement => (
  <Header.Title startIcon={icon}>{children}</Header.Title>
);
