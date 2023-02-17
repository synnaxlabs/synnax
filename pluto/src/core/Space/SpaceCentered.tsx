// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import clsx from "clsx";

import { Space, SpaceProps } from "./Space";

import "./SpaceCentered.css";

export type SpaceCenteredProps = SpaceProps;

export const SpaceCentered = ({
  className,
  justify = "center",
  align = "center",
  ...props
}: SpaceProps): JSX.Element => (
  <Space
    justify={justify}
    align={align}
    className={clsx("pluto-space-centered", className)}
    {...props}
  />
);
