// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, type ReactNode } from "react";

import { context } from "@/context";
import { type RenderProp, type Spec, type Tab } from "@/tabs/types";

export interface ContextValue {
  /** The array of tabs to display. */
  tabs: Tab[];
  /** Content to display when no tab is selected. */
  emptyContent?: ReactElement | null;
  /** Whether to display a close button on each tab. */
  closable?: boolean;
  /** The key of the currently selected tab. */
  selected?: string;
  /** A callback executed when a tab is selected. */
  onSelect?: (key: string) => void;
  /**
   * An optional render prop to display the content of the selected tab instead of using
   * the 'content' property of the tab object.
   */
  content?: RenderProp | ReactNode;
  /**
   * A callback executed when a tab is closed. The key of the tab to close is passed as
   * an argument. This callback is only executed if the tab is closable.
   */
  onClose?: (key: string) => void;
  /**
   * A callback executed when a tab's handle is dragged. Identical to a onDragStart
   * handler in react, except the tab object is passed as the second argument.
   *
   * @param e The drag event.
   * @param tab The tab being dragged.
   */
  onDragStart?: (e: React.DragEvent<HTMLElement>, tab: Spec) => void;
  /**
   * A callback executed when a tab's handle stops being dragged. Identical to a
   * onDragEnd handler in react, except the tab object is passed as the second argument.
   *
   * @param e The drag event.
   * @param tab The tab being dragged.
   */
  onDragEnd?: (e: React.DragEvent<HTMLElement>, tab: Spec) => void;
  /**
   * A callback executed when a tab is dropped. Identical to a onDrop handler in react.
   */
  onDrop?: (e: React.DragEvent<HTMLElement>) => void;
  /**
   * A callback executed when a tab is renamed.
   *
   * @param key the key of the tab to rename
   * @param name the name to rename the tab to
   */
  onRename?: (key: string, name: string) => void;
  /**
   * If provided, the Tabs.Selector component will display a button to create a new tab.
   * This callback is executed when that button is clicked.
   */
  onCreate?: () => void;
}

const [Context, useContext] = context.create<ContextValue>({
  defaultValue: { tabs: [] },
  displayName: "Tabs.Context",
});
export { useContext };

/**
 * Provider for the {@link Context} context. See the {@link ContextValue} type for
 * information on the shape of the context.
 */
export const Provider = Context;
