// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Triggers } from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactElement } from "react";

/** Root wraps a portal island with the providers Pluto's interactive pieces need. */
export const Root = ({ children }: PropsWithChildren): ReactElement => (
  <Triggers.Provider>{children}</Triggers.Provider>
);
