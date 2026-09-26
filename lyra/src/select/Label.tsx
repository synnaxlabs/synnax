// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import { type ReactNode } from "react";

import { CSS } from "@/css";
import { useIsFixed, useSlotRef } from "@/select/registry";

/** Props for {@link Label}. */
export interface LabelProps<K extends record.Key = record.Key> {
  itemKey: K | undefined;
  /** Shown when the key names no fixed {@link Item}. */
  children?: ReactNode;
}

/**
 * Shows the children of the fixed {@link Item} with the given key, or its own children
 * when the key names no fixed item. Triggers and tags use it to label a selection.
 */
export const Label = <K extends record.Key>({
  itemKey,
  children,
}: LabelProps<K>): ReactNode => {
  const fixed = useIsFixed(itemKey);
  const ref = useSlotRef(itemKey);
  if (!fixed) return children;
  return <span ref={ref} className={CSS.BE("select", "label")} />;
};
