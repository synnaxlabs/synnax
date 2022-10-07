import { useVirtualizer } from "@tanstack/react-virtual";
import { ComponentType, HTMLAttributes, Key, useRef } from "react";
import "./BaseList.css";
import { ListItemProps, TypedListEntry, useListContext } from "./ListContext";

export interface BaseListProps<K extends Key, E extends TypedListEntry<K>>
  extends Omit<HTMLAttributes<HTMLDivElement>, "children" | "onSelect"> {
  itemHeight: number;
  children: ComponentType<ListItemProps<K, E>>;
}

export default function VirtualCore<
  K extends Key,
  E extends TypedListEntry<K>
>({ itemHeight, children: Children, ...props }: BaseListProps<K, E>) {
  const {
    data,
    columnar: { columns },
    selected,
    onSelect,
  } = useListContext<K, E>();
  const parentRef = useRef<HTMLDivElement>(null);
  const virtualizer = useVirtualizer({
    count: data.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => itemHeight,
    overscan: Math.floor(data.length / 10),
  });
  return (
    <div
      ref={parentRef}
      className="pluto-list__container"
      style={{
        backgroundColor: "var(--pluto-gray-m3)",
        height: itemHeight * (data.length > 10 ? 10 : data.length),
      }}
      {...props}
    >
      <div
        className="pluto-list__inner"
        style={{ height: virtualizer.getTotalSize() }}
      >
        {virtualizer.getVirtualItems().map(({ index, start }, i) => {
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
}
