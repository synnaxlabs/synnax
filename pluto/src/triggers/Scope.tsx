// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PropsWithChildren, type ReactElement, useCallback } from "react";

import { context } from "@/context";

/** Condition is a boolean, or a getter read at the moment a trigger fires. */
export type Condition = boolean | (() => boolean);

export const resolveCondition = (cond: Condition): boolean =>
  typeof cond === "function" ? cond() : cond;

const [Context, useScope] = context.create<() => boolean>({
  defaultValue: () => true,
  displayName: "Triggers.Scope",
});
export { useScope };

export interface ScopeProps extends PropsWithChildren {
  active: Condition;
}

/**
 * Scope withholds trigger events from every {@link use} subscriber in its subtree while
 * active resolves false. Scopes nest: an inner scope cannot re-enable triggers that an
 * outer one has switched off.
 *
 * Keep active's identity stable. Every subscriber in the subtree re-renders when it
 * changes, so an inline arrow re-renders all of them on every render of this scope.
 */
export const Scope = ({ active, children }: ScopeProps): ReactElement => {
  const parent = useScope();
  const value = useCallback(
    () => parent() && resolveCondition(active),
    [parent, active],
  );
  return <Context value={value}>{children}</Context>;
};
