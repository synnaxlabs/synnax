// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/tree/Item.css";

import { type record } from "@synnaxlabs/x";
import { type MouseEvent } from "react";

import { type Button } from "@/button";
import { Caret } from "@/caret";
import { CSS } from "@/css";
import { Icon } from "@/icon";
import { Select } from "@/select";
import { useContext } from "@/tree/Context";

export type ItemProps<
  K extends record.Key,
  E extends Button.ElementType = "div",
> = Select.ItemProps<K, E> & {
  index: number;
  loading?: boolean;
};

const DEPTH_VARIABLE = CSS.variable("tree-item-depth");

export const Item = <K extends record.Key, E extends Button.ElementType = "div">({
  children,
  style,
  className,
  loading,
  ...rest
}: ItemProps<K, E>) => {
  const { index } = rest;
  const { nodes, toggle } = useContext("Tree.Item");
  const { expanded, depth, hasChildren } = nodes[index];
  const itemStyle = { [DEPTH_VARIABLE]: depth, ...style };
  return (
    <Select.Item
      className={CSS.cls(CSS.BE("tree", "item"), className)}
      style={itemStyle}
      gap="small"
      align="center"
      role="treeitem"
      aria-level={depth + 1}
      aria-expanded={hasChildren ? expanded : undefined}
      {...rest}
    >
      {hasChildren && (
        <Caret.Animated
          className={CSS.BE("tree", "expansion-indicator")}
          key="caret"
          enabled={expanded}
          enabledLoc="bottom"
          disabledLoc="right"
          onClick={
            toggle == null
              ? undefined
              : (e: MouseEvent) => {
                  e.stopPropagation();
                  toggle(index);
                }
          }
        />
      )}
      {children}
      {loading && <Icon.Loading />}
    </Select.Item>
  );
};
