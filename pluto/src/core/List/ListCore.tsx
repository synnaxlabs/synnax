// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { FunctionComponent, HTMLAttributes, useRef } from "react";

import { useVirtualizer } from "@tanstack/react-virtual";

import { useListContext } from "./ListContext";
import { ListItemProps } from "./types";

import { SelectedRecord } from "@/hooks/useSelectMultiple";
import { RenderableRecord } from "@/util/record";

import "./ListCore.css";

export interface ListVirtualCoreProps<E extends RenderableRecord<E>>
  extends Omit<HTMLAttributes<HTMLDivElement>, "children" | "onSelect"> {
  itemHeight: number;
  children: FunctionComponent<ListItemProps<E>>;
}

const ListVirtualCore = <E extends RenderableRecord<E>>({
  itemHeight,
  children,
  ...props
}: ListVirtualCoreProps<E>): JSX.Element => {
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
    overscan: Math.floor(data.length / 10),
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
