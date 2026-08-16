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
import { useMemo } from "react";

import { type Button } from "@/button";
import { Caret } from "@/caret";
import { CSS } from "@/css";
import { Icon } from "@/icon";
import { Select } from "@/select";
import { useContext } from "@/tree/Context";

export type ItemProps<
  K extends record.Key,
  E extends Button.ElementType = "div",
> = Select.ListItemProps<K, E> & {
  loading?: boolean;
  useMargin?: boolean;
  offsetMultiplier?: number;
};

export const Item = <K extends record.Key, E extends Button.ElementType = "div">({
  children,
  style,
  className,
  loading,
  useMargin = false,
  offsetMultiplier = 2.5,
  ...rest
}: ItemProps<K, E>) => {
  const { index } = rest;
  const { expanded, depth, hasChildren } = useContext("Tree.Item")[index];
  const itemStyle = useMemo(
    () => ({
      [CSS.var("tree-item-offset")]: `${depth * offsetMultiplier + 1.5}rem`,
      ...style,
    }),
    [depth, offsetMultiplier, style],
  );
  return (
    <Select.ListItem
      className={CSS(
        CSS.BE("tree", "item"),
        depth !== 0 && CSS.M("show-rules"),
        useMargin && CSS.M("margin"),
        className,
      )}
      style={itemStyle}
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
};
