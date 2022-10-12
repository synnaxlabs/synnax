import { useVirtualizer } from "@tanstack/react-virtual";
import { ComponentType, HTMLAttributes, Key, useRef } from "react";
import { useListContext } from "./ListContext";
import { ListItemProps, TypedListEntry } from "./Types";
import "./ListCore.css";

export interface ListVirtualCoreProps<
  K extends Key,
  E extends TypedListEntry<K>
> extends Omit<HTMLAttributes<HTMLDivElement>, "children" | "onSelect"> {
  itemHeight: number;
  children: ComponentType<ListItemProps<K, E>>;
}

const ListVirtualCore = <K extends Key, E extends TypedListEntry<K>>({
  itemHeight,
  children: Children,
  ...props
}: ListVirtualCoreProps<K, E>) => {
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
    <div ref={parentRef} className="pluto-list__container" {...props}>
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
};

const ListCore = {
  Virtual: ListVirtualCore,
};

export default ListCore;
