// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import { type ReactNode, useCallback } from "react";

import { CSS } from "@/css";
import { useIsFixed, useRegistryContext } from "@/select/registry";
import { Text } from "@/text";

/** Props for {@link Label}. */
export interface LabelProps<K extends record.Key = record.Key> {
  itemKey: K | undefined;
  /** The label element's id, for a trigger's aria-labelledby. */
  id?: string;
  /** Shown when the key names no fixed {@link Item}. */
  children?: ReactNode;
}

/**
 * Shows the children of the fixed {@link Item} with the given key, or its own children
 * when the key names no fixed item. Triggers and tags use it to label a selection. The
 * label shrinks and fades at its end when its row runs out of space.
 */
export const Label = <K extends record.Key>({
  itemKey,
  id,
  children,
}: LabelProps<K>): ReactNode => {
  const fixed = useIsFixed(itemKey);
  const registry = useRegistryContext("Select.Label");
  const ref = useCallback(
    (host: HTMLElement | null) => {
      if (host == null || itemKey == null) return;
      const label = registry.getLabel(itemKey);
      host.appendChild(label);
      return () => label.remove();
    },
    [registry, itemKey],
  );
  return (
    <Text.Text
      el="span"
      id={id}
      overflow="fade"
      className={CSS.BE("select", "label")}
      ref={fixed ? ref : undefined}
    >
      {fixed ? null : children}
    </Text.Text>
  );
};
