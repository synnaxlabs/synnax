// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Breadcrumb } from "@synnaxlabs/lyra/breadcrumb";
import { Header } from "@synnaxlabs/lyra/header";
import { type ReactElement } from "react";

const GAP = 1.5;

export interface TitleProps extends Pick<Header.TitleProps, "children"> {}

export const Title = ({ children }: TitleProps): ReactElement => (
  <Header.Title color={10} weight={500} gap={GAP}>
    {children}
  </Header.Title>
);

export interface TitleSegmentProps extends Pick<Breadcrumb.SegmentProps, "children"> {}

/** A {@link Title} for a toolbar whose header is a breadcrumb. */
export const TitleSegment = ({ children }: TitleSegmentProps): ReactElement => (
  <Breadcrumb.Segment color={10} weight={500} level="h5" gap={GAP}>
    {children}
  </Breadcrumb.Segment>
);
