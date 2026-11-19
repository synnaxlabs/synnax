// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Badge } from "@synnaxlabs/lyra/badge";
import { type ReactElement } from "react";

import { Channel } from "@/channel";
import { Grid } from "@/schematic/node/common/grid";

/** Marks a symbol whose channel the Core deleted, below the symbol. */
export const Item = Grid.createItem((): ReactElement => (
  <Grid.Item itemKey="deleted" location="bottom">
    <Badge.Badge>{Channel.DELETED_TEXT}</Badge.Badge>
  </Grid.Item>
));
