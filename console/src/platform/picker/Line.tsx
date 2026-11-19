// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/picker/Line.css";

import { Flex } from "@synnaxlabs/lyra/flex";
import { type PropsWithChildren, type ReactElement } from "react";

import { CSS } from "@/platform/css";

/** A line under a select, such as the state of its entry, inset to its trigger. */
export const Line = ({ children }: PropsWithChildren): ReactElement => (
  <Flex.Box x align="center" className={CSS.BE("picker", "line")}>
    {children}
  </Flex.Box>
);
