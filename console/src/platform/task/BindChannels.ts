// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { useEffect } from "react";

import { useIsPreview } from "@/platform/task/Form";

export interface BindChannelsProps<C extends record.Keyed<string>> {
  /** The form path of the list. Defaults to `config.channels`. */
  path?: string;
  /**
   * Returns the bindings the device holds for an entry. Return null to leave the entry
   * as it is, and for every entry while the device is unknown. A binding of 0 or one
   * the entry already carries is dropped, so a resolver may return the device's
   * primitive bindings unconditionally; one that builds a nested value must return
   * null when it is unchanged.
   */
  resolve: (entry: C) => Partial<C> | null;
}

/**
 * Keeps every entry of a task form's channel list bound to the Synnax channel its
 * device already maps to the entry's port. An entry for a mapped port then shows that
 * channel, and a rename reaches the Core instead of a name the deploy discards. Runs
 * again as entries are added or edited. An unmapped port never unbinds an entry: a
 * deploy binds the config before the device record catches up. A preview form is left
 * alone. Renders nothing.
 */
export const BindChannels = <C extends record.Keyed<string>>({
  path = "config.channels",
  resolve,
}: BindChannelsProps<C>): null => {
  const { set } = Form.useContext();
  const state = Form.useFieldState<C[]>(path, { optional: true });
  const isPreview = useIsPreview();
  useEffect(() => {
    if (isPreview || state == null) return;
    let changed = false;
    const next = state.value.map((entry) => {
      const patch = resolve(entry);
      if (patch == null) return entry;
      const keys = (Object.keys(patch) as (keyof C)[]).filter(
        (k) => patch[k] !== 0 && patch[k] !== entry[k],
      );
      if (keys.length === 0) return entry;
      changed = true;
      const bound = { ...entry };
      keys.forEach((k) => (bound[k] = patch[k] as C[keyof C]));
      return bound;
    });
    if (changed) set(path, next);
  }, [state, resolve, isPreview, path, set]);
  return null;
};
