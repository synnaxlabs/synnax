// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type base } from "@/flux/base";
import { hashQuery } from "@/flux/base/queryCache";
import { successResult } from "@/flux/result";
import { createRetrieve, type UseRetrieve } from "@/flux/suspense/retrieve";
import { state } from "@/state";

export interface CreateSelectorParams<
  Query extends base.Shape,
  ParentData extends base.Shape,
  ParentStore extends base.Store,
  ParentAllowDisconnected extends boolean,
  Selected extends base.Shape,
> {
  /// The parent retrieve hook this selector derives from. The selector
  /// suspends if the parent isn't loaded, and re-derives the slice when the
  /// parent's listeners push updates.
  upstream: UseRetrieve<Query, ParentData, ParentStore, ParentAllowDisconnected>;
  /// The slice derivation. Receives the loaded parent value and the selector's
  /// query, returns the slice.
  select: (parent: ParentData, query: Query) => Selected;
  /// Value-equality check on the derived slice. When the new slice equals the
  /// previous one, the cache skips the notification and subscribers don't
  /// re-render. This is the optimization that makes selectors avoid re-renders
  /// on unrelated parent changes.
  equal?: (a: Selected, b: Selected) => boolean;
  /// Optional name suffix appended to the parent's name, used to scope the
  /// selector's cache hash apart from the parent's. Defaults to "slice".
  name?: string;
}

/// Builds a suspending selector hook. A selector is a derived retrieve: it
/// ensures the parent record is loaded (suspending if not), then returns a
/// slice of the parent.
///
/// Both the selector and its parent have cache entries. The parent's entry
/// holds the full record; the selector's entry holds the derived slice. The
/// selector's listener mounts the parent's listeners (so the parent stays
/// fresh under realtime updates) and subscribes to the parent's cache hash
/// (so slice re-derivation runs whenever the parent value changes). The
/// selector's cache entry is set with the slice's `equal` check, so the
/// consumer only re-renders when the slice itself differs.
export const createSelector = <
  Query extends base.Shape,
  ParentData extends base.Shape,
  ParentStore extends base.Store,
  ParentAllowDisconnected extends boolean,
  Selected extends base.Shape,
>(
  params: CreateSelectorParams<
    Query,
    ParentData,
    ParentStore,
    ParentAllowDisconnected,
    Selected
  >,
): UseRetrieve<Query, Selected, ParentStore, ParentAllowDisconnected> => {
  const { upstream, select, equal, name = "slice" } = params;
  const { spec: parentSpec } = upstream;

  return createRetrieve<Query, Selected, ParentStore, ParentAllowDisconnected>({
    name: `${parentSpec.name}:${name}`,
    allowDisconnected: parentSpec.allowDisconnected,
    equal,
    retrieve: async (retrieveParams) => {
      const parentValue = await parentSpec.retrieve(retrieveParams);
      // Populate the parent's cache entry so listener-driven setter updates
      // can resolve against it later.
      const parentHash = hashQuery({
        name: parentSpec.name,
        query: retrieveParams.query,
      });
      retrieveParams.cache.set(parentHash, successResult(parentSpec.name, parentValue));
      return select(parentValue, retrieveParams.query);
    },
    mountListeners: ({ cache, query, onChange, ...rest }) => {
      const parentHash = hashQuery({ name: parentSpec.name, query });

      // Bridge from parent listener to parent cache: resolve setter-style
      // updates against the parent's current cached value, then store the
      // result. This keeps the parent cache the canonical source of the
      // parent record across listener updates.
      const parentOnChange = (value: state.SetArg<ParentData | undefined>): void => {
        const parentEntry = cache.get<ParentData>(parentHash);
        const prev = parentEntry?.variant === "success" ? parentEntry.data : undefined;
        const nextParent = state.executeSetter(value, prev);
        if (nextParent == null) return;
        cache.set(parentHash, successResult(parentSpec.name, nextParent));
      };

      // Subscribe to the parent's cache so slice derivation runs whenever the
      // parent changes. The selector's `onChange` writes the slice into the
      // selector cache, where the consumer's subscription lives.
      const cacheSub = cache.subscribe(parentHash, () => {
        const entry = cache.get<ParentData>(parentHash);
        if (entry?.variant !== "success") return;
        onChange(select(entry.data, query));
      });

      const parentDestructors = parentSpec.mountListeners?.({
        ...rest,
        cache,
        query,
        onChange: parentOnChange,
      });
      const arr =
        parentDestructors == null
          ? []
          : Array.isArray(parentDestructors)
            ? parentDestructors
            : [parentDestructors];
      return [...arr, cacheSub];
    },
  });
};
