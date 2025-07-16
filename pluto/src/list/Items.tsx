// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/list/Items.css";

import { type record } from "@synnaxlabs/x";
import { memo, type ReactElement } from "react";

import { Align } from "@/align";
import { CSS } from "@/css";
import { useData } from "@/list/Frame";
import { type ItemRenderProp } from "@/list/Item";

export interface ItemsProps<K extends record.Key = record.Key>
  extends Omit<Align.SpaceProps, "children" | "ref"> {
  children: ItemRenderProp<K>;
  emptyContent?: ReactElement;
}

const BaseItems = <
  K extends record.Key = record.Key,
  E extends record.Keyed<K> | undefined = record.Keyed<K>,
>({
  className,
  children,
  emptyContent,
  ...rest
}: ItemsProps<K>): ReactElement => {
  const { ref, getItems, getTotalSize, data } = useData<K, E>();
  const visibleData = getItems();
  let content = emptyContent;
  if (data.length > 0)
    content = (
      <div
        className={CSS.BE("list", "virtualizer")}
        style={{ minHeight: getTotalSize() }}
      >
        {visibleData.map(({ key, index, translate }) =>
          children({ key, index, itemKey: key, translate }),
        )}
      </div>
    );
  return (
    <Align.Space
      empty
      ref={ref}
      className={CSS(className, CSS.BE("list", "items"))}
      {...rest}
    >
      {content}
    </Align.Space>
  );
};

export const Items = memo(BaseItems) as typeof BaseItems;
