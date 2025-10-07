// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Header, type Icon } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

export interface TitleProps extends Pick<Header.TitleProps, "children"> {
  icon: Icon.ReactElement;
}

export const Title = ({ icon, children }: TitleProps): ReactElement => (
  <Header.Title color={10} weight={500}>
    {icon}
    {children}
  </Header.Title>
);
