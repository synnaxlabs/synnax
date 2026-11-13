// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Icon } from "@synnaxlabs/lyra/icon";
import { type FC } from "react";

/** Props for the picker of a {@link Property}. */
export interface PickerProps<V> {
  /** The condition on the property, or null for none. */
  value: V | null;
  /** Takes the new condition, or null to drop it. */
  onChange: (value: V | null) => void;
}

/** Props for the parts of a chip that show a condition. */
export interface ConditionProps<V> {
  value: V;
  onChange: (value: V | null) => void;
}

/**
 * A property that a view filters by, such as labels. Each holds at most one condition
 * of the filter.
 */
export interface Property<W, V> {
  name: string;
  icon: Icon.FC;
  /** @returns the condition a node of the filter holds, or null for another node. */
  decode: (node: W) => V | null;
  encode: (value: V) => W;
  /**
   * Picks the condition: a selection that opens on the menu surface. The filter menu
   * opens it beside the menu, and a chip opens it below the chip.
   */
  Picker: FC<PickerProps<V>>;
  /** The part of a chip that compares the property to the value, such as "is". */
  Operator: FC<ConditionProps<V>>;
  /** The value in a chip, which opens the picker. */
  Value: FC<ConditionProps<V>>;
}

/** The properties a view filters by, keyed by a name for each. */
export type Properties<W> = Record<string, Property<W, any>>;
