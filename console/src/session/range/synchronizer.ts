// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot, NotFoundError } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/lyra/status";

import { toRef } from "@/session/range/ref";
import {
  type Action,
  markStaticKeysRewritten,
  remove,
  type StoreState,
} from "@/session/range/slice";
import { type Synchronizer } from "@/session/synchronizer";

const syncRanges: Synchronizer.Callbacks<StoreState, Action> = {
  reconcile: async ({ client, store }) => {
    // Only the Core's own ranges can vanish; the rest are the session's.
    const keys = store
      .getState()
      .range.ranges.filter(({ variant }) => variant === "persisted")
      .map(({ key }) => key);
    if (keys.length === 0) return;
    const found = await client.ranges.retrieve({ key: keys });
    const foundKeys = new Set(found.map(({ key }) => key));
    const missing = keys.filter((key) => !foundKeys.has(key));
    if (missing.length > 0) store.dispatch(remove({ keys: missing }));
  },
  // A rename or a retime needs no answer here: the session keeps only the key, and
  // reads the rest from the Core.
  listen: ({ client, store }) =>
    client.ranges.onDelete((key) => store.dispatch(remove({ keys: [key] }))),
};

const REWRITE_FAILED = { name: "Line plot", message: "Failed to update static ranges" };

// Desktop 1.0 stored a static range's key on a line plot. The Core cannot resolve the
// key, so the plot shows the range's window instead. A plot deleted meanwhile counts as
// done. A plot the user cannot update, or a failed send, leaves the rewrite to a later
// pass, as only this session holds its static ranges.
const useRewriteStaticKeys = (): Synchronizer.Callbacks<StoreState, Action> => {
  const handleError = Status.useErrorHandler();
  return {
    reconcile: async ({ client, store, ended }) => {
      const { ranges, staticKeysRewritten } = store.getState().range;
      if (staticKeysRewritten) return;
      const windows = new Map(
        ranges
          .filter(({ variant }) => variant === "static")
          .map((s) => [s.key, toRef(s)]),
      );
      if (windows.size === 0) {
        if (!ended()) store.dispatch(markStaticKeysRewritten());
        return;
      }
      const grantee = await client.users.current.retrieve();
      const plots = await client.lineplots.retrieve({});
      let skipped = false;
      const errors: unknown[] = [];
      await Promise.all(
        plots.map(async ({ key, ranges }) => {
          const actions = ranges.flatMap((range) => {
            const window = range.variant === "key" ? windows.get(range.key) : undefined;
            if (window == null) return [];
            return [
              lineplot.removeRange({ range }),
              lineplot.addRange({ range: window }),
            ];
          });
          if (actions.length === 0) return;
          const updatable = await client.access.granted.retrieve(grantee, {
            objects: lineplot.resourceID(key),
            action: "update",
          });
          if (!updatable) {
            skipped = true;
            return;
          }
          try {
            await client.lineplots.send(key, actions);
          } catch (e) {
            if (!NotFoundError.matches(e)) errors.push(e);
          }
        }),
      );
      if (ended()) return;
      if (errors.length > 0) handleError(errors[0], REWRITE_FAILED);
      else if (!skipped) store.dispatch(markStaticKeysRewritten());
    },
  };
};

export const SYNCHRONIZERS: Synchronizer.Synchronizers<StoreState, Action> = [
  { name: "sync ranges", use: () => syncRanges },
  { name: "rewrite static range keys", use: useRewriteStaticKeys },
];
