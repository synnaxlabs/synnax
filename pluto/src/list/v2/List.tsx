import { type record } from "@synnaxlabs/x";
import { useVirtualizer } from "@tanstack/react-virtual";
import { type ComponentPropsWithoutRef, type FC, useRef, useState } from "react";

import { CSS } from "@/css";
import { type RenderProp } from "@/util/renderProp";

interface ItemProps<K extends record.Key> {
  itemKey: K;
}

export type Item<K extends record.Key> = FC<ItemProps<K>>;

export interface VirtualItemsProps<K extends record.Key = record.Key>
  extends Omit<ComponentPropsWithoutRef<"div">, "children"> {
  data: K[];
  itemHeight: number;
  children: RenderProp<ItemProps<K>>;
  overscan?: number;
}

export const Items = <K extends record.Key>({
  data,
  className,
  itemHeight,
  children,
}: VirtualItemsProps<K>) => {
  const virtualizer = useVirtualizer({
    count: data.length,
    getScrollElement: () => document.body,
    estimateSize: () => itemHeight,
  });
  const items = virtualizer.getVirtualItems();
  const parentRef = useRef<HTMLDivElement>(null);
  return (
    <div ref={parentRef} className={CSS(className, CSS.BE("list", "container"))}>
      <div
        className={CSS.BE("list", "virtualizer")}
        style={{ height: virtualizer.getTotalSize() }}
      >
        {items.map(({ index }) => children({ itemKey: data[index] }))}
      </div>
    </div>
  );
};

const Example = () => {
  const [keys, setKeys] = useState<record.Key[]>([]);
  const [sourceData, setSourceData] = useState<
    Map<record.Key, Record<string, unknown>>
  >(new Map());
};
