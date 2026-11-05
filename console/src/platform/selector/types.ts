// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel } from "@synnaxlabs/client";
import { type Icon } from "@synnaxlabs/lyra/icon";

/** OnSelectParams is passed to a selectable's onSelect hook when it is chosen. */
export interface OnSelectParams {
  /**
   * tabKey is the selector tab the pick was made from, so the selectable can open
   * its content into that tab in place rather than a new one.
   */
  tabKey?: panel.TabKey;
}

/** An item a selector offers to create. The button the selector draws calls its hooks. */
export interface Selectable {
  /** Identifies the item, such as a resource or task type. */
  type: string;
  /** Names the item on a card, and for search and screen readers. */
  title: string;
  /** A shorter name for a button under a group heading, such as "Analog read". */
  label?: string;
  /** A short line under the title on a card. */
  description?: string;
  icon: Icon.ReactElement;
  /** Returns the function that creates the item when it is picked. */
  useOnSelect: (params?: OnSelectParams) => () => void;
  /** Returns false to leave the item out, such as for a user who may not create it. */
  useVisible?: () => boolean;
}

/** A column of buttons for one integration. */
export interface Group {
  key: string;
  title: string;
  icon: Icon.ReactElement;
  selectables: Selectable[];
}

/** A part of a selector: a row of cards, or columns of groups. */
export interface Section {
  key: string;
  title?: string;
  /** Items drawn as cards. */
  selectables?: Selectable[];
  /** Items drawn as buttons, in one column per group. */
  groups?: Group[];
  /** Returns false to leave the section out. */
  useVisible?: () => boolean;
}
