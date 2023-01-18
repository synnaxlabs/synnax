// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { HTMLAttributes, useRef } from "react";

import { RenderableRecord } from "@synnaxlabs/x";
import { useVirtualizer } from "@tanstack/react-virtual";

import { SelectedRecord } from "@/hooks/useSelectMultiple";
import { RenderProp } from "@/util/renderProp";

import { useListContext } from "./ListContext";
import { ListItemProps } from "./types";

import "./ListCore.css";

export interface ListVirtualCoreProps<E extends RenderableRecord<E>>
  extends Omit<HTMLAttributes<HTMLDivElement>, "children" | "onSelect"> {
  itemHeight: number;
  children: RenderProp<ListItemProps<E>>;
  overscan?: number;
}

const ListVirtualCore = <E extends RenderableRecord<E>>({
  itemHeight,
  children,
  overscan = 5,
  ...props
}: ListVirtualCoreProps<E>): JSX.Element => {
  if (itemHeight <= 0) throw new Error("itemHeight must be greater than 0");
  const {
    data,
    columnar: { columns },
    select: { onSelect },
  } = useListContext<E>();
  const parentRef = useRef<HTMLDivElement>(null);
  const virtualizer = useVirtualizer({
    count: data.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => itemHeight,
    overscan,
  });
  return (
    <div ref={parentRef} className="pluto-list__container" {...props}>
      <div className="pluto-list__inner" style={{ height: virtualizer.getTotalSize() }}>
        {virtualizer.getVirtualItems().map(({ index, start }) => {
          const entry = data[index];
          return children({
            key: index,
            index,
            onSelect,
            entry,
            columns,
            selected: (entry as SelectedRecord<E>)?.selected ?? false,
            style: {
              transform: `translateY(${start}px)`,
              position: "absolute",
            },
          });
        })}
      </div>
    </div>
  );
};

export const ListCore = {
  Virtual: ListVirtualCore,
};
