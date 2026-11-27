// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { channel } from "@synnaxlabs/client";
import { Haul } from "@synnaxlabs/lyra/haul";
import { DataType } from "@synnaxlabs/x";

export const HAUL_TYPE = "channel";

/** The fields of a channel that decide whether a line plot can draw it. */
export type PlottableFields = Pick<
  channel.Channel,
  "dataType" | "virtual" | "expression"
>;

/**
 * Returns whether a line plot can draw the channel: its samples are numbers, and it has
 * a time axis, so it stores samples or is calculated.
 */
export const plottable = ({
  dataType,
  virtual,
  expression,
}: PlottableFields): boolean =>
  dataType.isNumeric &&
  !dataType.equals(DataType.TIMESTAMP) &&
  (!virtual || expression !== "");

/** Whether a line plot can draw the channel, when the drag source knows it. */
export interface HaulData {
  plottable: boolean;
}

export type HaulItem = Haul.Item<typeof HAUL_TYPE, channel.Key, HaulData | undefined>;

/**
 * Creates a haul item for the channel with the key. Passing the channel records whether
 * a line plot can draw it, so that drop targets in any window can tell.
 */
export const createHaulItem = (key: channel.Key, ch?: PlottableFields): HaulItem => ({
  type: HAUL_TYPE,
  key,
  data: ch == null ? undefined : { plottable: plottable(ch) },
});

export const isHaulItem = (item: Haul.Item): item is HaulItem =>
  item.type === HAUL_TYPE;

export const filterHaulItems = (items: Haul.Item[]): HaulItem[] =>
  items.filter(isHaulItem);

export const canDropHaulItem = Haul.canDropOfType<HaulItem>(HAUL_TYPE);

/**
 * Returns the channel items a line plot can draw. An item whose source did not know
 * the channel counts as plottable.
 */
export const filterPlottableHaulItems = (items: Haul.Item[]): HaulItem[] =>
  filterHaulItems(items).filter(({ data }) => data?.plottable !== false);
