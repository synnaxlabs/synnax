// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/tree/Item.css";

import { type record } from "@synnaxlabs/x";

import { type Button } from "@/button";
import { Caret } from "@/caret";
import { CSS } from "@/css";
import { Icon } from "@/icon";
import { type List } from "@/list";
import { Select } from "@/select";
import { type ItemRenderProps } from "@/tree/Tree";

export type ItemProps<
  K extends record.Key,
  E extends Button.ElementType = "div",
> = ItemRenderProps<K> &
  Select.ListItemProps<K, E> & {
    loading?: boolean;
    useMargin?: boolean;
    offsetMultiplier?: number;
  };

export const Item = <K extends record.Key, E extends Button.ElementType = "div">({
  depth,
  hasChildren,
  expanded,
  children,
  style,
  className,
  loading,
  useMargin = false,
  offsetMultiplier = 2.5,
  ...rest
}: ItemProps<K, E>) => (
  // @ts-expect-error - generic element issues
  <Select.ListItem<K, E>
    className={CSS(
      CSS.BE("tree", "item"),
      depth !== 0 && CSS.M("show-rules"),
      useMargin && CSS.M("margin"),
      className,
    )}
    style={{
      [CSS.var("tree-item-offset")]: `${depth * offsetMultiplier + 1.5}rem`,
      ...style,
    }}
    gap="small"
    align="center"
    {...rest}
  >
    {hasChildren && (
      <Caret.Animated
        className={CSS.BE("tree", "expansion-indicator")}
        key="caret"
        enabled={expanded}
        enabledLoc="bottom"
        disabledLoc="right"
      />
    )}
    {children}
    {loading && <Icon.Loading />}
  </Select.ListItem>
);
