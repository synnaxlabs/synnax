// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useEffect } from "react";

import { box, type xy } from "@synnaxlabs/x";

import { Aether } from "@/aether";
import { Theming } from "@/theming";
import { type Tooltip as Core } from "@/tooltip";
import { Table } from "@/vis/table";

export interface TooltipProps extends Core.ContentProps {
  position: xy.XY;
  sources: Table.SourceProps[];
}

export const Tooltip = Aether.wrap<TooltipProps>(
  "Tooltip",
  ({ aetherKey, sources, position, triggerDims, ...props }): ReactElement => {
    const [, , setState] = Table.use({ aetherKey, sources, region: box.ZERO, size: 1 });
    useEffect(() => {
      setState((p) => ({
        ...p,
        region: box.construct(
          position.x - 100 + triggerDims.width / 2 + theme.sizes.base,
          position.y - theme.sizes.base * (sources.length * 2 + 3),
          200,
          200,
        ),
      }));
    }, [position]);
    const theme = Theming.use();
    return (
      <div
        style={{ height: sources.length * (theme.sizes.base * 2 + 2), width: 200 }}
      />
    );
  },
);
