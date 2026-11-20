// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { context } from "@synnaxlabs/lyra/context";
import { Store } from "@synnaxlabs/lyra/store";
import {
  type PropsWithChildren,
  type ReactElement,
  type RefCallback,
  useState,
} from "react";

// The channel under the pointer, in a channel list or in its map. Both mark it, so each
// points at the other.
const Hovered = Store.createPresence("Task.Port.Hovered");
const [SetHovered, useSetHoveredContext] = context.create<(key?: string) => void>({
  defaultValue: () => {},
  displayName: "Task.Port.SetHovered",
});

/**
 * Shares the channel under the pointer between a channel list and its map. Outside it,
 * no channel is ever hovered.
 */
export const HoverProvider = ({ children }: PropsWithChildren): ReactElement => {
  const [hovered, setHovered] = useState<string>();
  return (
    <SetHovered value={setHovered}>
      <Hovered.Context value={hovered}>{children}</Hovered.Context>
    </SetHovered>
  );
};

/** @returns the key of the channel under the pointer, in the list or in the map. */
export const useHovered = Hovered.usePresent;

/** @returns whether the channel is under the pointer, in the list or in the map. */
export const useIsHovered = Hovered.useIsPresent;

/** @returns a function that marks the channel with the key as hovered, or none. */
export const useSetHovered = useSetHoveredContext;

/**
 * @returns a ref that marks the channel as hovered while the pointer is over the
 * element. It listens to native events, since React also sends the element the pointer
 * events of its portals, such as an open menu.
 */
export const useHoveredRef = (key: string): RefCallback<HTMLElement> => {
  const setHovered = useSetHovered();
  return (el) => {
    if (el == null) return;
    const handleEnter = () => setHovered(key);
    const handleLeave = () => setHovered();
    el.addEventListener("pointerenter", handleEnter);
    el.addEventListener("pointerleave", handleLeave);
    return () => {
      el.removeEventListener("pointerenter", handleEnter);
      el.removeEventListener("pointerleave", handleLeave);
    };
  };
};
