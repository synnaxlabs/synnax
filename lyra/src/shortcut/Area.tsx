// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { destructor } from "@synnaxlabs/x";
import {
  type PropsWithChildren,
  type ReactElement,
  type RefObject,
  useEffect,
  useState,
} from "react";

import { context } from "@/context";
import { useSyncedRef } from "@/hooks/ref";

/** A part of the page that provides commands. Areas nest. */
export interface Node {
  parent: Node | null;
  depth: number;
  active: RefObject<boolean>;
}

/** Registers a handler for a command in an area, returning the unregister. */
export type Provide = (
  command: string,
  node: Node,
  handler: RefObject<() => void>,
) => destructor.Destructor;

export interface AreaContextValue {
  node: Node;
  provide: Provide;
}

const [AreaContext, useAreaContext] = context.create<AreaContextValue>({
  displayName: "Shortcut.Area",
  providerName: "Shortcut.Provider",
});
export { AreaContext };

/** @returns whether the node and every area around it are active. */
export const isActive = (node: Node | null): boolean =>
  node == null || (node.active.current && isActive(node.parent));

/** Props for {@link Area}. */
export interface AreaProps extends PropsWithChildren {
  active: boolean;
}

/**
 * Provides the commands of its subtree while active is true and every area around it
 * is active. A key runs the command of the deepest active area that provides one for
 * it. At equal depth, the area mounted last wins.
 */
export const Area = ({ active, children }: AreaProps): ReactElement => {
  const parent = useAreaContext("Shortcut.Area");
  const activeRef = useSyncedRef(active);
  const [value] = useState<AreaContextValue>(() => ({
    node: { parent: parent.node, depth: parent.node.depth + 1, active: activeRef },
    provide: parent.provide,
  }));
  return <AreaContext value={value}>{children}</AreaContext>;
};

/**
 * Runs handler when a key bound to the command is pressed, or when the command runs
 * from {@link useContext}, while the enclosing area is the one that wins it.
 */
export const useCommand = (command: string, handler: () => void): void => {
  const { node, provide } = useAreaContext("Shortcut.useCommand");
  const handlerRef = useSyncedRef(handler);
  useEffect(() => provide(command, node, handlerRef), [command, node, provide]);
};
