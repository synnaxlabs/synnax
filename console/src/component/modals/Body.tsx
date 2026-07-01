// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/component/modals/Body.css";

import { Flex } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { CSS } from "@/component/css";

export interface BodyProps extends Flex.BoxProps {}

export const Body = ({ className, ...rest }: BodyProps): ReactElement => (
  <Flex.Box
    y
    grow
    justify="center"
    className={CSS(CSS.BE("modal", "body"), className)}
    {...rest}
  />
);
