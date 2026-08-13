// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/shell/Shell.css";

import { Logo } from "@synnaxlabs/media";
import { Flex } from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactElement } from "react";

import { CSS } from "@/platform/css";

/**
 * The ringed Synnax icon every pre-workspace surface centers on its card. Pass
 * children to put something else inside the ring.
 */
export const Mark = ({ children }: PropsWithChildren): ReactElement => (
  <Flex.Box
    y
    empty
    align="center"
    justify="center"
    grow={false}
    className={CSS.BE("shell", "mark-ring")}
  >
    {children ?? <Logo variant="icon" className={CSS.BE("shell", "mark")} />}
  </Flex.Box>
);
