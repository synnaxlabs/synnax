// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useState } from "react";

/** Draft is the state of a field that edits a value and sends it to a target. */
export interface Draft<V> {
  /** value is the value the field shows, or null when none is known. */
  value: V | null;
  /** dirty is true while the field shows a value the target does not report. */
  dirty: boolean;
  /**
   * edit stages a value for the field to show. Null, or the value the field shows
   * without an edit, discards the edit.
   */
  edit: (value: V | null) => void;
  /**
   * send marks the shown value as sent. The field keeps showing it until the target
   * reports a new value, so a write that fails stays visible and dirty.
   */
  send: () => void;
}

interface Sent<V> {
  value: V;
  target: V | null;
}

/**
 * useDraft tracks a field that edits and sends a value to a target, such as a command
 * channel. Without an edit or a pending send, the field follows the target.
 *
 * @param target is the value the target reports, or null when none is known.
 */
export const useDraft = <V>(target: V | null): Draft<V> => {
  const [edited, setEdited] = useState<V | null>(null);
  const [sent, setSent] = useState<Sent<V> | null>(null);
  // Any new report settles a send, since the target can hold a value other than the
  // one sent, such as a rounded number. A set during render, behind a guard, makes
  // React retry before it commits.
  const settled = sent != null && (target !== sent.target || target === sent.value);
  if (settled) setSent(null);
  const base = (settled ? null : sent?.value) ?? target;
  // An edit that matches the base is dropped, so the field keeps following the target.
  if (edited != null && edited === base) setEdited(null);
  const value = edited ?? base;
  return {
    value,
    dirty: value !== target,
    edit: setEdited,
    send: () => {
      if (value != null) setSent({ value, target });
      setEdited(null);
    },
  };
};
