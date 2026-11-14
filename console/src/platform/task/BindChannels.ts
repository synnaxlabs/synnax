// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, query } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Channel } from "@synnaxlabs/pluto";
import { deep, record, unique } from "@synnaxlabs/x";
import { useEffect, useRef } from "react";

import { useIsPreview } from "@/platform/task/useIsPreview";

export interface BindChannelsProps<C extends record.Keyed<string>> {
  /** The form path of the list. */
  path: deep.Target<C[]>;
  /**
   * Returns the bindings the device maps for an entry, or null while the device is
   * unknown. Each number a binding changes is a channel key. A zero binding, or a key
   * of a channel that does not exist, unbinds the entry only when the device mapped it
   * before.
   */
  resolve: (entry: C) => Partial<C> | null;
  /**
   * Changes each bound entry to agree with the channels it binds. It runs on every
   * pass, so it must return the entry itself when nothing changes.
   */
  match?: (entry: C, channels: channel.Channel[]) => C;
}

const isPlainObject = (value: unknown): value is object =>
  typeof value === "object" && value !== null && !Array.isArray(value);

/**
 * @returns the channel keys that binding the patch to the entry would set. When
 * unchanged is true, it also returns each key the entry already holds.
 */
const boundKeys = (entry: object, patch: object, unchanged: boolean): channel.Key[] =>
  Object.entries(patch).flatMap(([k, value]) => {
    const current: unknown = (entry as Record<string, unknown>)[k];
    if (isPlainObject(value) && isPlainObject(current))
      return boundKeys(current, value, unchanged);
    return typeof value === "number" && value !== 0 && (unchanged || value !== current)
      ? [value]
      : [];
  });

/** Sets each key the patch would bind that is not in found to zero. */
const dropMissing = <T extends object>(
  entry: T,
  patch: Partial<T>,
  found: ReadonlySet<channel.Key>,
): Partial<T> => {
  const out: Record<string, unknown> = {};
  for (const [k, value] of Object.entries(patch)) {
    const current: unknown = (entry as Record<string, unknown>)[k];
    if (isPlainObject(value) && isPlainObject(current))
      out[k] = dropMissing(current, value, found);
    else if (typeof value === "number" && value !== current && !found.has(value))
      out[k] = 0;
    else out[k] = value;
  }
  return out as Partial<T>;
};

/**
 * Applies the bindings the device maps to an entry, descending into nested objects.
 * Returns the same entry when nothing changes.
 * @param before - The bindings the device last mapped for the entry.
 */
const bind = <T extends object>(
  entry: T,
  patch: Partial<T>,
  before: Partial<T> | undefined,
): T => {
  const changes: Partial<T> = {};
  let changed = false;
  for (const k of record.keys(patch)) {
    const value = patch[k];
    const current = entry[k];
    const prior = before?.[k];
    if (value === undefined) continue;
    if (isPlainObject(value) && isPlainObject(current)) {
      const nested = bind(current, value, isPlainObject(prior) ? prior : undefined);
      if (nested === current) continue;
      changes[k] = nested;
    } else {
      if (deep.equal(value, current)) continue;
      // A device record trailing a deploy maps nothing yet, so a zero clears only a
      // binding the device gave before.
      if (value === 0 && (prior == null || prior === 0)) continue;
      changes[k] = value;
    }
    changed = true;
  }
  return changed ? { ...entry, ...changes } : entry;
};

/**
 * Binds each entry of a task form's channel list to the channel its device already
 * maps, so the entry shows that channel and a rename reaches the Core. A channel the
 * device maps but the Core no longer holds is not bound. Runs again as entries are
 * added or edited, once the mapped channels are retrieved. A preview form is left
 * alone. Renders nothing.
 */
export const BindChannels = <C extends record.Keyed<string>>({
  path,
  resolve,
  match,
}: BindChannelsProps<C>): null => {
  const { set } = Form.useContext();
  const state = Form.useFieldState(path, { optional: true });
  const isPreview = useIsPreview();
  const held = useRef(new Map<string, Partial<C>>());
  const entries = isPreview || state == null ? null : state.value;
  const patches = entries?.map(resolve) ?? [];
  const keys = unique.unique(
    patches.flatMap((patch, i) =>
      patch == null || entries == null
        ? []
        : boundKeys(entries[i], patch, match != null),
    ),
  );
  const { data: channels } = Channel.useResultMultiple(
    keys.length === 0 ? null : query.keysRequest(keys),
  );
  const found = keys.length === 0 ? [] : channels?.map(({ key }) => key);
  useEffect(() => {
    if (entries == null || found == null) return;
    const live = new Set(found);
    const next = new Map<string, Partial<C>>();
    const bound = entries.map((entry, i) => {
      const resolved = patches[i];
      const before = held.current.get(entry.key);
      if (resolved == null) {
        if (before != null) next.set(entry.key, before);
        return entry;
      }
      const patch = dropMissing(entry, resolved, live);
      next.set(entry.key, patch);
      const bound = bind(entry, patch, before);
      return match == null ? bound : match(bound, channels ?? []);
    });
    held.current = next;
    if (bound.some((entry, i) => entry !== entries[i])) set(path, bound);
  }, [entries, patches, found, channels, match, path, set]);
  return null;
};
