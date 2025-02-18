// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/tabs/Tabs.css";

import { direction } from "@synnaxlabs/x";
import React, {
  createContext,
  type ReactElement,
  type ReactNode,
  use,
  useCallback,
} from "react";

import { Align } from "@/align";
import { CSS } from "@/css";
import { useSyncedRef } from "@/hooks";
import { state } from "@/state";
import { Selector, type SelectorProps } from "@/tabs/Selector";
import { type RenderProp, type Spec, type Tab } from "@/tabs/types";
import { type ComponentSize } from "@/util/component";

/**
 * Checks if the selected tab key exists in the tabs array. If it does not, it returns
 * the last tab key in the array. If the array is empty, it returns undefined. This
 * function is useful for 'resetting' the selected tab when a tab is removed that may
 * be the currently selected tab.
 *
 * @param selected The currently selected tab key.
 * @param tabs The array of tabs to search through.
 * @returns The selected tab key or undefined if the array is empty.
 */
export const resetSelection = (selected = "", tabs: Tab[] = []): string | undefined => {
  if (tabs.length === 0) return undefined;
  return tabs.find((t) => t.tabKey === selected) != null
    ? selected
    : tabs[tabs.length - 1]?.tabKey;
};

/**
 * Finds the tab with the given key and renames it to the given name, shallowly copying
 * the array and the tab object.
 * @param key The key of the tab to rename.
 * @param name The new name of the tab.
 * @param tabs The array of tabs to search through.
 * @returns A new array of tabs with the renamed tab.
 */
export const rename = (key: string, name: string, tabs: Tab[]): Tab[] => {
  name = name.trim();
  if (name.length === 0) return tabs;
  const t = tabs.find((t) => t.tabKey === key);
  if (t == null || t.name === name) return tabs;
  return tabs.map((t) => (t.tabKey === key ? { ...t, name } : t));
};

/** Props for the {@link useStatic} hook. */
export interface UseStaticTabsProps {
  tabs: Tab[];
  content?: RenderProp;
  onSelect?: (key: string) => void;
  selected?: string;
}

/**
 * A utility hook for creating a static set of tabs. This hook is useful when the list
 * of tabs you'd like to display and switch between is known ahead of time and is not
 * dynamically adjusted.
 *
 * @param tabs The array of tabs to display.
 * @param content An optional render prop to display the content of the selected tab
 * instead of pulling from the 'content' property of the tab object.
 * @pram onSelect An optional callback to be called when a tab is selected.
 * @param selected The key of the tab to be selected by default.
 * @returns props to pass to the {@link Tabs} component.
 */
export const useStatic = ({
  tabs,
  content,
  selected,
  onSelect,
}: UseStaticTabsProps): ContextValue => {
  const [value, onChange] = state.usePurePassthrough({
    initial: selected ?? tabs[0]?.tabKey ?? "",
    value: selected,
    onChange: onSelect,
  });
  const valueRef = useSyncedRef(selected ?? value);

  const handleSelect = useCallback(
    (key: string): void => {
      onChange(key);
      if (valueRef.current == null) onSelect?.(key);
    },
    [value, onSelect],
  );

  return {
    tabs,
    selected: value,
    content,
    onSelect: handleSelect,
  };
};

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
  onDragStart?: (e: React.DragEvent<HTMLDivElement>, tab: Spec) => void;
  /**
   * A callback executed when a tab's handle stops being dragged. Identical to a
   * onDragEnd handler in react, except the tab object is passed as the second argument.
   *
   * @param e The drag event.
   * @param tab The tab being dragged.
   */
  onDragEnd?: (e: React.DragEvent<HTMLDivElement>, tab: Spec) => void;
  /**
   * A callback executed when a tab is dropped. Identical to a onDrop handler in react.
   */
  onDrop?: (e: React.DragEvent<HTMLDivElement>) => void;
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

/** Props for the {@link Tabs} component. */
export interface TabsProps
  extends Omit<
      Align.SpaceProps,
      | "children"
      | "onSelect"
      | "size"
      | "onDragStart"
      | "onDragEnd"
      | "content"
      | "contextMenu"
    >,
    ContextValue {
  children?: RenderProp | ReactNode;
  size?: ComponentSize;
  selectedAltColor?: boolean;
  contextMenu?: SelectorProps["contextMenu"];
}

/**
 * Context used to propagate tab related information to children. See the {@link ContextValue}
 * type for information on the shape of the context.
 */
const Context = createContext<ContextValue>({ tabs: [] });

/**
 * Provider for the {@link Context} context. See the {@link ContextValue} type for information
 * on the shape of the context.
 */
export const Provider = Context;

/** @returns The current value of the {@link Context} context. */
export const useContext = () => use(Context);

/**
 * High-level component for creating a tabbed interface. This component is a composition
 * of the {@link Selector}, {@link Content}, and {@link Context} components to provide a
 * complete tabbed interface. It's also possible to use these components individually
 * to create a custom tabbed interface.
 *
 * @param content Optional render prop to display the content of the selected tab instead
 * of using the 'content' property of the tab object. This can be a function or a React
 * element.
 * @param children The same as the 'content' prop, but as a child element. If this prop
 * is specified, it will take precedence over the 'content' prop.
 * @param onSelect A callback executed when a tab is selected. The key of the selected tab
 * is passed as an argument.
 * @param selected The key of the currently selected tab.
 * @param selectedAltColor Whether to use an alternate color for the selected tab.
 * @param closable Whether to display a close button on each tab.
 * @param tabs The array of tabs to display.
 * @param onClose A callback executed when a tab is closed. The key of the tab to close is
 * passed as an argument. This callback is only executed if the tab is closable.
 * @param onDragStart A callback executed when a tab's handle is dragged. Identical to a
 * onDragStart handler in react, except the tab object is passed as the second argument.
 * @param onDragEnd A callback executed when a tab's handle stops being dragged. Identical
 * to a onDragEnd handler in react, except the tab object is passed as the second argument.
 * @param onCreate If provided, the Tabs.Selector component will display a button to create
 * a new tab. This callback is executed when that button is clicked.
 * @param onRename A callback executed when a tab is renamed.
 * @param emptyContent Content to display when no tab is selected.
 * @param className An optional class name to apply to the component.
 * @param onDragOver A callback executed when a tab is dragged over the component. Identical
 * to a onDragOver handler in react.
 * @param onDrop A callback executed when a tab is dropped. Identical to a onDrop handler in
 * react.
 * @param size The size of the tabs selector to display. Can be "small", "medium", or "large".
 * @note all other props are inherited from the {@link Align.Space} component and are passed
 * through to that component.
 * @param direction The direction in which to show the tabs selector. An 'x' direction
 * will show the selector on the left side of the tabs, while a 'y' direction will show
 * the selector on the top side of the tabs.
 */
export const Tabs = ({
  id,
  content,
  children,
  onSelect,
  selected,
  selectedAltColor,
  closable,
  tabs,
  onClose,
  onDragStart,
  onDragEnd,
  onCreate,
  onRename,
  emptyContent,
  className,
  onDragOver,
  onDrop,
  contextMenu,
  size = "medium",
  direction: dir = "y",
  ...rest
}: TabsProps): ReactElement => (
  <Align.Space
    id={id}
    empty
    className={CSS(CSS.B("tabs"), className)}
    onDragOver={onDragOver}
    onDrop={onDrop}
    direction={dir}
    {...rest}
  >
    <Provider
      value={{
        tabs,
        emptyContent,
        selected,
        closable,
        content: children ?? content,
        onSelect,
        onClose,
        onDragStart,
        onDragEnd,
        onRename,
        onCreate,
        onDrop,
      }}
    >
      <Selector
        size={size}
        direction={direction.swap(dir)}
        altColor={selectedAltColor}
        contextMenu={contextMenu}
      />
      <Content />
    </Provider>
  </Align.Space>
);

export const Content = (): ReactNode | null => {
  const { tabs, selected, content: renderProp, emptyContent, onSelect } = useContext();
  let content: ReactNode = null;
  const selectedTab = tabs.find((tab) => tab.tabKey === selected);
  if (selected == null || selectedTab == null) return emptyContent ?? null;
  if (renderProp != null)
    if (typeof renderProp === "function") content = renderProp(selectedTab);
    else content = renderProp;
  else if (selectedTab.content != null) content = selectedTab.content as ReactNode;
  return (
    <div className={CSS.B("tabs-content")} onClick={() => onSelect?.(selected)}>
      {content}
    </div>
  );
};
