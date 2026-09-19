// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/pluto";
import { deep, type record } from "@synnaxlabs/x";
import { useEffect, useRef } from "react";

import { useIsPreview } from "@/platform/task/Form";

export interface BindChannelsProps<C extends record.Keyed<string>> {
  /** The form path of the list. Defaults to `config.channels`. */
  path?: string;
  /**
   * Returns the bindings the device holds for an entry. Return null to leave the entry
   * as it is, and for every entry while the device is unknown. A binding the entry
   * already carries is dropped, so a resolver may return the device's bindings
   * unconditionally.
   */
  resolve: (entry: C) => Partial<C> | null;
}

/**
 * Keeps every entry of a task form's channel list bound to the Synnax channel its
 * device already maps to the entry's port. An entry for a mapped port then shows that
 * channel, and a rename reaches the Core instead of a name the deploy discards. Runs
 * again as entries are added or edited, so editing a port carries its entry to the
 * channel the new port maps, or clears it when the new port maps nothing. A preview
 * form is left alone. Renders nothing.
 */
export const BindChannels = <C extends record.Keyed<string>>({
  path = "config.channels",
  resolve,
}: BindChannelsProps<C>): null => {
  const { set } = Form.useContext();
  const state = Form.useFieldState<C[]>(path, { optional: true });
  const isPreview = useIsPreview();
  // The bindings the device last held for each entry. A zero only clears a binding
  // the device mapped before, so an edit that moves an entry to an unmapped port
  // unbinds it while a device record trailing a deploy leaves it alone.
  const held = useRef(new Map<string, Partial<C>>());
  useEffect(() => {
    if (isPreview || state == null) return;
    const nextHeld = new Map<string, Partial<C>>();
    let changed = false;
    const next = state.value.map((entry) => {
      const patch = resolve(entry);
      const before = held.current.get(entry.key);
      if (patch == null) {
        if (before != null) nextHeld.set(entry.key, before);
        return entry;
      }
      nextHeld.set(entry.key, patch);
      const keys = (Object.keys(patch) as (keyof C)[]).filter((k) =>
        patch[k] === 0
          ? before != null && before[k] !== 0 && before[k] != null && entry[k] !== 0
          : !deep.equal(patch[k], entry[k]),
      );
      if (keys.length === 0) return entry;
      changed = true;
      const bound = { ...entry };
      keys.forEach((k) => (bound[k] = patch[k] as C[keyof C]));
      return bound;
    });
    held.current = nextHeld;
    if (changed) set(path, next);
  }, [state, resolve, isPreview, path, set]);
  return null;
};
