// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/common/indicator/Indicator.css";

import { Badge } from "@synnaxlabs/lyra/badge";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import type { ReactElement } from "react";

import { CSS } from "@/css";
import { Grid } from "@/schematic/node/common/grid";
import { Staleness } from "@/vis/staleness";

/** Unknown shows that a state of a symbol has no data, or names no entry. */
const Unknown = (): ReactElement => (
  <Badge.Badge
    status="warning"
    className={CSS.B("unknown-badge")}
    role="img"
    aria-label="Unknown"
  >
    <Status.Indicator variant="warning">
      <Icon.QuestionMark className={CSS.BE("unknown-badge", "icon")} />
    </Status.Indicator>
  </Badge.Badge>
);

export interface BadgesProps {
  stale: boolean;
  unknown: boolean;
}

/** Badges shows the indicators of a symbol side by side, or nothing. */
export const Badges = ({ stale, unknown }: BadgesProps): ReactElement | null => {
  if (!stale && !unknown) return null;
  return (
    <Flex.Box x align="center" gap="tiny">
      {stale && <Staleness.Badge />}
      {unknown && <Unknown />}
    </Flex.Box>
  );
};

/** Slot shows the indicators of a symbol in its bottom slot, or nothing. */
export const Slot = Grid.createItem(({ stale, unknown }: BadgesProps) => {
  if (!stale && !unknown) return null;
  return (
    <Grid.Item itemKey="indicator" location="bottom">
      <Badges stale={stale} unknown={unknown} />
    </Grid.Item>
  );
});
Slot.displayName = "Indicator.Slot";
