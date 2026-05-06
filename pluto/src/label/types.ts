// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Haul } from "@synnaxlabs/charon";
import { type label } from "@synnaxlabs/client";

export const HAUL_TYPE = "label";

export type HaulItem = Haul.Item<typeof HAUL_TYPE, label.Key, undefined>;

export const createHaulItem = (key: label.Key): HaulItem => ({
  type: HAUL_TYPE,
  key,
});

export const isHaulItem = (item: Haul.Item): item is HaulItem =>
  item.type === HAUL_TYPE;

export const filterHaulItems = (items: Haul.Item[]): HaulItem[] =>
  items.filter(isHaulItem);

export const canDropHaulItem = Haul.canDropOfType<HaulItem>(HAUL_TYPE);
