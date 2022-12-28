import type { ComponentType, CSSProperties } from "react";

import { RenderableRecord } from "@/util/record";

type RenderF<E extends RenderableRecord<E> = RenderableRecord> = ComponentType<{
  entry: E;
  style: CSSProperties;
}>;

export interface ListColumn<E extends RenderableRecord<E> = RenderableRecord> {
  key: keyof E;
  render?: RenderF<E>;
  label: string;
  visible?: boolean;
  width?: number;
}

export interface ListItemProps<E extends RenderableRecord<E>> {
  key: string | number;
  entry: E;
  index: number;
  style: React.CSSProperties;
  selected: boolean;
  columns: Array<ListColumn<E>>;
  onSelect?: (key: string) => void;
}
