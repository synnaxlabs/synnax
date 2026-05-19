// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Haul } from "@/haul";
import { type AddNodeProps } from "@/schematic/queries";

export const HAUL_TYPE = "schematic-element";

export interface HaulItemData extends AddNodeProps {}

export type HaulItem = Haul.Item<typeof HAUL_TYPE, string, HaulItemData>;

export const createHaulItem = (data: HaulItemData): HaulItem => ({
  type: HAUL_TYPE,
  key: data.key,
  data,
});

export const isHaulItem = (item: Haul.Item): item is HaulItem =>
  item.type === HAUL_TYPE;

export const filterHaulItems = (items: Haul.Item[]): HaulItem[] =>
  items.filter(isHaulItem);

export const canDropHaulItem: Haul.CanDrop = Haul.canDropOfType(HAUL_TYPE);
