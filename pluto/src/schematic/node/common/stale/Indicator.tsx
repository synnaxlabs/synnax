// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Grid } from "@/schematic/node/common/grid";
import { Staleness } from "@/vis/staleness";

/** Indicator shows the stale indicator in the bottom slot of a symbol. */
export const Indicator = Grid.createItem(() => (
  <Grid.Item itemKey="stale" location="bottom">
    <Staleness.Badge />
  </Grid.Item>
));
Indicator.displayName = "Stale.Indicator";
