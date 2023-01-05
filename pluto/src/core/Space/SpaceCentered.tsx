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

export interface SpaceCenteredProps extends Omit<SpaceProps, "justify" | "align"> {}

export const SpaceCentered = ({ className, ...props }: SpaceProps): JSX.Element => (
  <Space
    {...props}
    justify="center"
    align="center"
    className={clsx("pluto-space-centered", className)}
  />
);
