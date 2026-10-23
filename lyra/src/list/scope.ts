// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import { type RefCallback, useLayoutEffect } from "react";

import { context } from "@/context";

/** Records the element of a mounted row, or null when the row unmounts. */
export type SetElement = (key: record.Key, element: HTMLElement | null) => void;

/** Acts on the mounted items of a Frame by key. */
export interface Elements {
  setElement: SetElement;
  /** Clicks the mounted item with the given key. */
  click: (key: record.Key) => void;
  /** Keeps the item with the given key mounted when it scrolls out of view. */
  pin: (key: record.Key | null) => void;
}

export const [ElementsContext, useElementsContext] = context.create<Elements>({
  displayName: "List.ElementsContext",
  providerName: "List.Frame",
});

/** @returns a function that clicks the item of the enclosing Frame by key. */
export const useClick = (): Elements["click"] =>
  useElementsContext("List.useClick").click;

/**
 * Keeps the item with the given key mounted while it is out of view, so it can still be
 * clicked. Pass undefined to release it.
 */
export const usePin = (key: record.Key | undefined): void => {
  const { pin } = useElementsContext("List.usePin");
  useLayoutEffect(() => {
    pin(key ?? null);
    return () => pin(null);
  }, [pin, key]);
};

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
