// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Align, Header } from "@synnaxlabs/pluto";
import { type ReactNode } from "react";

import { CSS } from "@/css";

export interface ChannelDetailsProps {
  headerActions?: Header.ActionsProps["children"];
  children: ReactNode;
}

export const ChannelDetails = ({ children, headerActions }: ChannelDetailsProps) => (
  <Align.Space direction="y" grow>
    <Header.Header level="h4">
      <Header.Title weight={500} wrap={false}>
        Details
      </Header.Title>
      {headerActions != null && <Header.Actions>{headerActions}</Header.Actions>}
    </Header.Header>
    <Align.Space className={CSS.B("details")}>{children}</Align.Space>
  </Align.Space>
);
