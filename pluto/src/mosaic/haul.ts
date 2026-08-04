// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";

import { Haul } from "@/haul";

/**
 * This type should be used when the user wants to drop a tab in the mosaic.
 * Dropping an item with this signature will call the {@link Frame} onDrop handler.
 */
export const HAUL_DROP_TYPE = "pluto_mosaic_tab_drop";

export type TabDropHaulItem = Haul.Item<
  typeof HAUL_DROP_TYPE,
  string,
  record.Unknown | undefined
>;

/**
 * @param data - Opaque payload attached by the drag source and handed back to the
 * {@link Frame} onDrop handler. A mosaic in another window is a separate drop target
 * with its own state, so anything the destination needs about the tab's origin travels
 * here.
 */
export const createTabDropHaulItem = (
  tabKey: string,
  elementID?: string,
  data?: record.Unknown,
): TabDropHaulItem => ({ type: HAUL_DROP_TYPE, key: tabKey, elementID, data });

export const isTabDropHaulItem = (item: Haul.Item): item is TabDropHaulItem =>
  item.type === HAUL_DROP_TYPE;

export const filterTabDropHaulItems = (items: Haul.Item[]): TabDropHaulItem[] =>
  items.filter(isTabDropHaulItem);

export const canDropTabDropHaulItem =
  Haul.canDropOfType<TabDropHaulItem>(HAUL_DROP_TYPE);

/** This type should be used when the user wants to create a new tab in the mosaic.
Dropping an item with this signature will call the {@link Frame} onCreate handler. */
export const HAUL_CREATE_TYPE = "pluto_mosaic_tab_create";

export type TabCreateHaulItem = Haul.Item<typeof HAUL_CREATE_TYPE, string, undefined>;

export const createTabCreateHaulItem = (tabKey: string): TabCreateHaulItem => ({
  type: HAUL_CREATE_TYPE,
  key: tabKey,
});

export const isTabCreateHaulItem = (item: Haul.Item): item is TabCreateHaulItem =>
  item.type === HAUL_CREATE_TYPE;

export const filterTabCreateHaulItems = (items: Haul.Item[]): TabCreateHaulItem[] =>
  items.filter(isTabCreateHaulItem);

export const canDropTabCreateHaulItem =
  Haul.canDropOfType<TabCreateHaulItem>(HAUL_CREATE_TYPE);
