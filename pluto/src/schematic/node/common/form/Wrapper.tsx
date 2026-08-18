// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/common/form/form.css";

import { type FC, type ReactElement } from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";

interface WrapperProps extends Flex.BoxProps {}

export const Wrapper: FC<WrapperProps> = ({
  className,
  direction,
  ...rest
}): ReactElement => (
  <Flex.Box
    direction={direction}
    align="stretch"
    className={CSS.cx(CSS.B("symbol-form"), className)}
    gap={direction === "x" ? "large" : "medium"}
    {...rest}
  />
);
