// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel } from "@synnaxlabs/client";
import { type Haul } from "@synnaxlabs/pluto";

/** Haul type of a panel dragged by its pill in the selector strip. */
export const PILL_HAUL_TYPE = "console_panel_pill";

export type PillHaulItem = Haul.Item<typeof PILL_HAUL_TYPE, panel.Key, undefined>;

export const createPillHaulItem = (key: panel.Key): PillHaulItem => ({
  type: PILL_HAUL_TYPE,
  key,
});

export const isPillHaulItem = (item: Haul.Item): item is PillHaulItem =>
  item.type === PILL_HAUL_TYPE;
