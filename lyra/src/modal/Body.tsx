// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/modal/Body.css";

import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";

export interface BodyProps extends Flex.BoxProps {}

/** Body is the scrolling middle of a modal, between its {@link Header} and
 * {@link Footer}. */
export const Body = ({ className, ...rest }: BodyProps): ReactElement => (
  <Flex.Box y grow className={CSS.cls(CSS.BE("modal", "body"), className)} {...rest} />
);
