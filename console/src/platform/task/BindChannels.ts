// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/pluto";
import { deep, record } from "@synnaxlabs/x";
import { useEffect, useRef } from "react";

import { useIsPreview } from "@/platform/task/Form";

export interface BindChannelsProps<C extends record.Keyed<string>> {
  /** The form path of the list. Defaults to `config.channels`. */
  path?: string;
  /**
   * Returns the bindings the device maps for an entry, or null while the device is
   * unknown. A zero binding unbinds the entry only when the device mapped it before.
   */
  resolve: (entry: C) => Partial<C> | null;
}

/**
 * Applies the bindings the device maps to an entry. Returns the same entry when
 * nothing changes.
 * @param before - The bindings the device last mapped for the entry.
 */
const bind = <C extends record.Keyed<string>>(
  entry: C,
  patch: Partial<C>,
  before: Partial<C> | undefined,
): C => {
  const changes: Partial<C> = {};
  let changed = false;
  for (const k of record.keys(patch)) {
    const value = patch[k];
    if (value === undefined || deep.equal(value, entry[k])) continue;
    // A device record trailing a deploy maps nothing yet, so a zero clears only a
    // binding the device gave before.
    const prior = before?.[k];
    if (value === 0 && (prior == null || prior === 0)) continue;
    changes[k] = value;
    changed = true;
  }
  return changed ? { ...entry, ...changes } : entry;
};

/**
 * Binds each entry of a task form's channel list to the channel its device already
 * maps, so the entry shows that channel and a rename reaches the Core. Runs again as
 * entries are added or edited. A preview form is left alone. Renders nothing.
 */
export const BindChannels = <C extends record.Keyed<string>>({
  path = "config.channels",
  resolve,
}: BindChannelsProps<C>): null => {
  const { set } = Form.useContext();
  const state = Form.useFieldState<C[]>(path, { optional: true });
  const isPreview = useIsPreview();
  const held = useRef(new Map<string, Partial<C>>());
  useEffect(() => {
    if (isPreview || state == null) return;
    const entries = state.value;
    const next = new Map<string, Partial<C>>();
    const bound = entries.map((entry) => {
      const patch = resolve(entry);
      const before = held.current.get(entry.key);
      if (patch == null) {
        if (before != null) next.set(entry.key, before);
        return entry;
      }
      next.set(entry.key, patch);
      return bind(entry, patch, before);
    });
    held.current = next;
    if (bound.some((entry, i) => entry !== entries[i])) set(path, bound);
  }, [state, resolve, isPreview, path, set]);
  return null;
};
