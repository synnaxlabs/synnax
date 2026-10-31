// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { type PropsWithChildren, type ReactElement } from "react";

/**
 * Root wraps a portal island with the providers Pluto's interactive pieces need. A page
 * renders one Root, which also shows the full text of every cut label on the page.
 */
export const Root = ({ children }: PropsWithChildren): ReactElement => (
  <Triggers.Provider>
    <Tooltip.Config>
      <Tooltip.Overflow />
      {children}
    </Tooltip.Config>
  </Triggers.Provider>
);
