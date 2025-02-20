// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Header, type Icon } from "@synnaxlabs/pluto";

export interface TitleProps extends Pick<Header.TitleProps, "children"> {
  icon: Icon.Element;
}

export const Title = ({ icon, children }: TitleProps) => (
  <Header.Title shade={8} startIcon={icon} weight={500}>
    {children}
  </Header.Title>
);
