// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";

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

// Desktop 1.0 stored a static range's key on a line plot. The Core cannot resolve the
// key, so the plot shows the range's window instead.
const rewriteStaticKeys: Synchronizer.Callbacks<StoreState, Action> = {
  reconcile: async ({ client, store }) => {
    const { ranges, staticKeysRewritten } = store.getState().range;
    if (staticKeysRewritten) return;
    const windows = new Map(
      ranges
        .filter(({ variant }) => variant === "static")
        .map((s) => [s.key, toRef(s)]),
    );
    if (windows.size > 0) {
      const plots = await client.lineplots.retrieve({});
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
          if (actions.length > 0) await client.lineplots.send(key, actions);
        }),
      );
    }
    store.dispatch(markStaticKeysRewritten());
  },
};

export const SYNCHRONIZERS: Synchronizer.Synchronizers<StoreState, Action> = [
  { name: "sync ranges", use: () => syncRanges },
  { name: "rewrite static range keys", use: () => rewriteStaticKeys },
];
