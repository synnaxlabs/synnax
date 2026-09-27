// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import { type RefCallback } from "react";

import { context } from "@/context";

/** Records the element of a mounted row, or null when the row unmounts. */
export type SetElement = (key: record.Key, element: HTMLElement | null) => void;

/**
 * Holds the element setter of the nearest Frame under an Items render function, and
 * null elsewhere.
 */
export const [ItemsContext, useItemsContext] = context.create<SetElement | null>({
  defaultValue: null,
  displayName: "List.ItemsContext",
});

/** @returns true under an Items render function, for the rows of the nearest Frame. */
export const useInItems = (): boolean => useItemsContext() != null;

/** Holds the scroll ref of the Frame whose Scroll encloses the subtree. */
export const [ScrollContext, useScrollContext] = context.create<
  RefCallback<HTMLDivElement | null>
>({
  displayName: "List.ScrollContext",
  providerName: "List.Scroll",
});
