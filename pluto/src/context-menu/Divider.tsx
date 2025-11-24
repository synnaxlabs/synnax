// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/context-menu/Divider.css";

import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Divider as CoreDivider } from "@/divider";

export const Divider = ({
  className,
  ...rest
}: CoreDivider.DividerProps): ReactElement => (
  <CoreDivider.Divider
    x
    padded
    className={CSS(CSS.BE("menu", "divider"), className)}
    {...rest}
  />
);
