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

import { Caret } from "@/caret";
import { CSS } from "@/css";
import { Select } from "@/select";
import { type ItemProps } from "@/tree/Tree";

export const Item = <K extends record.Key>({
  depth,
  hasChildren,
  expanded,
  children,
  style,
  className,
  showRules = true,
  ...rest
}: ItemProps<K>) => (
  <Select.ListItem
    className={CSS(
      CSS.BE("tree", "item"),
      showRules && depth !== 0 && CSS.M("show-rules"),
      className,
    )}
    style={{
      [CSS.var("tree-item-offset")]: `${depth * 2.5 + 1.5}rem`,
      ...style,
    }}
    size="small"
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
  </Select.ListItem>
);
