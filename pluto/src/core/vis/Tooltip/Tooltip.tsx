// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useEffect } from "react";

import { z } from "zod";

import { Aether } from "@/core/aether/main";
import { AetherTooltip } from "@/core/vis/Tooltip/aether";

export interface TooltipProps extends z.input<typeof AetherTooltip.stateZ> {}

export const Tooltip = Aether.wrap<TooltipProps>(
  "Tooltip",
  ({ position, aetherKey }): ReactElement | null => {
    const [, , setState] = Aether.use({
      aetherKey,
      type: AetherTooltip.TYPE,
      schema: AetherTooltip.stateZ,
      initialState: {
        position,
      },
    });

    useEffect(() => setState({ position }), [position]);

    return null;
  }
);
