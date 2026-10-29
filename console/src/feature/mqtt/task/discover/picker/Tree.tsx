// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component } from "@synnaxlabs/lyra/component";
import { context } from "@synnaxlabs/lyra/context";
import { type CSS as PCSS } from "@synnaxlabs/lyra/css";
import { Flex } from "@synnaxlabs/lyra/flex";
import { useSyncedRef } from "@synnaxlabs/lyra/hooks";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Skeleton } from "@synnaxlabs/lyra/skeleton";
import { Text } from "@synnaxlabs/lyra/text";
import { Tree as Base } from "@synnaxlabs/lyra/tree";
import { deep } from "@synnaxlabs/x";
import { type ReactElement, useLayoutEffect } from "react";

import { CheckSkeleton } from "@/feature/mqtt/task/discover/picker/Picker";
import { CSS } from "@/platform/css";

// The skeleton items match it.
const ITEM_HEIGHT = 33;

/** What one item of a {@link Tree} shows. */
export interface TreeItem {
  key: string;
  name: string;
  /** Trails the name: a count, or how many values are picked. */
  meta: string;
  /** How much of what it holds is picked. Absent on an item with no checkbox. */
  picked?: "none" | "some" | "all";
  disabled?: boolean;
  loading?: boolean;
  /** A click folds or unfolds it as well. */
  folder?: boolean;
}

const [PickContext, usePickContext] = context.create<(key: string) => void>({
  displayName: "Picker.PickContext",
  providerName: "Picker.Tree",
});

const Item = (props: Base.ItemRenderProps<string>): ReactElement | null => {
  const { itemKey } = props;
  const item = List.useItem<string, TreeItem>(itemKey);
  const pick = usePickContext("Picker.Tree.Item");
  if (item == null) return null;
  const { name, meta, picked, disabled, loading } = item;
  const accented = picked === "some" || picked === "all";
  return (
    <Base.Item {...props} loading={loading}>
      {picked != null && (
        <Input.Checkbox
          value={picked === "all"}
          indeterminate={picked === "some"}
          onChange={() => pick(itemKey)}
          disabled={disabled}
          size="small"
          aria-label={`Pick ${name}`}
        />
      )}
      <Text.Text level="small" weight={500} color={10} variant="code" overflow="fade">
        {name}
      </Text.Text>
      <Text.Text
        level="small"
        color={accented ? undefined : 9}
        className={CSS.cls(
          CSS.BE("mqtt-discover", "meta"),
          accented && CSS.M("picked"),
        )}
      >
        {meta}
      </Text.Text>
    </Base.Item>
  );
};

const item = Component.renderProp(Item);

export interface TreeProps extends Pick<
  Base.UseProps<string>,
  "nodes" | "initialExpanded"
> {
  /** What each node shows, by its key. */
  items: TreeItem[];
  /** The key of the item whose values the pane shows. */
  active: string | null;
  onActivate: (key: string) => void;
  onPick: (key: string) => void;
  emptyContent: ReactElement;
}

/**
 * A tree of things to pick, each with a checkbox. A change to the items re-renders
 * only the items whose content changed.
 */
export const Tree = ({
  nodes,
  initialExpanded,
  items,
  active,
  onActivate,
  onPick,
  emptyContent,
}: TreeProps): ReactElement => {
  const store = List.useMapData<string, TreeItem>({ initialData: items });
  useLayoutEffect(() => {
    const changed = items.filter((i) => !deep.equal(store.getItem(i.key), i));
    if (changed.length > 0) store.setItem(changed);
  }, [items, store]);
  const tree = Base.use({ nodes, initialExpanded, toggleOn: "caret" });
  const { toggle } = tree;
  const handleSelect: Base.TreeProps<string, TreeItem>["onSelect"] = (
    _,
    { clicked },
  ) => {
    if (clicked == null) return;
    onActivate(clicked);
    if (store.getItem(clicked)?.folder === true) toggle(clicked);
  };
  // Stable, so a new onPick re-renders no item.
  const pickRef = useSyncedRef(onPick);
  const pick = (key: string) => pickRef.current(key);
  return (
    <PickContext value={pick}>
      <Base.Tree<string, TreeItem>
        {...tree}
        getItem={store.getItem}
        subscribe={store.subscribe}
        selected={active == null ? [] : [active]}
        onSelect={handleSelect}
        itemHeight={ITEM_HEIGHT}
        emptyContent={emptyContent}
      >
        {item}
      </Base.Tree>
    </PickContext>
  );
};

/** A skeleton item in a tree: its depth and the width of its name. */
export type SkeletonItem = [depth: number, width: string];

const SKELETON_STYLE: PCSS.VarProperties = {
  "--console-item-height": `${ITEM_HEIGHT}px`,
};

const indent = (depth: number): PCSS.VarProperties => ({ "--console-depth": depth });

export interface TreeSkeletonProps {
  items: SkeletonItem[];
}

/** Holds the place of a {@link Tree} while the first listen runs. */
export const TreeSkeleton = ({ items }: TreeSkeletonProps): ReactElement => (
  <div
    aria-hidden
    className={CSS.BE("mqtt-discover", "skeleton")}
    style={SKELETON_STYLE}
  >
    {items.map(([depth, width], i) => (
      <Flex.Box
        key={i}
        x
        align="center"
        gap="small"
        style={indent(depth)}
        className={CSS.BE("mqtt-discover", "skeleton-item")}
      >
        <CheckSkeleton />
        <Text.Text level="small">
          <Skeleton.Skeleton style={{ width }} />
        </Text.Text>
      </Flex.Box>
    ))}
  </div>
);
