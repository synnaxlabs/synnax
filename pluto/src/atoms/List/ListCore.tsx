// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ComponentType, HTMLAttributes, useRef } from "react";

import { useVirtualizer } from "@tanstack/react-virtual";

import { useListContext } from "./ListContext";
import { RenderableRecord, ListItemProps } from "./types";
import "./ListCore.css";

export interface ListVirtualCoreProps<E extends RenderableRecord<E>>
  extends Omit<HTMLAttributes<HTMLDivElement>, "children" | "onSelect"> {
  itemHeight: number;
  children: ComponentType<ListItemProps<E>>;
}

const ListVirtualCore = <E extends RenderableRecord<E>>({
  itemHeight,
  children: Children,
  ...props
}: ListVirtualCoreProps<E>): JSX.Element => {
  const {
    data,
    columnar: { columns },
    selected,
    onSelect,
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
          return (
            <Children
              key={`${entry.key}`}
              index={index}
              onSelect={onSelect}
              entry={entry}
              columns={columns}
              selected={selected.includes(entry.key)}
              style={{
                transform: `translateY(${start}px)`,
                position: "absolute",
              }}
            />
          );
        })}
      </div>
    </div>
  );
};

export const ListCore = {
  Virtual: ListVirtualCore,
};
