// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/search/Bar.css";

import { Flex } from "@synnaxlabs/lyra/flex";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";

export interface BarProps extends Flex.BoxProps {}

/** The bordered strip above a list, for a search input and its actions. */
export const Bar = ({ className, ...rest }: BarProps): ReactElement => (
  <Flex.Box
    x
    bordered
    background={1}
    justify="between"
    align="center"
    className={CSS.cls(CSS.BE("search", "bar"), className)}
    {...rest}
  />
);
