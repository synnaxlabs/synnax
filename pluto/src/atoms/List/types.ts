import type { ComponentType, CSSProperties } from "react";

interface ListColumn {
  label: string;
  visible?: boolean;
  width?: number;
}

type RenderF<E extends RenderableRecord<E> = RenderableRecord> = ComponentType<{
  entry: E;
  style: CSSProperties;
}>;

export type UntypedListColumn = {
  key: string;
  render?: RenderF;
} & ListColumn;

export type TypedListColumn<E extends RenderableRecord<E>> = {
  key: keyof E;
  render?: RenderF<E>;
} & ListColumn;

export type RenderableRecord<E = Record<string, string | number | undefined>> = {
  key: string;
} & Partial<Record<keyof E, string | number | undefined>>;

export type UntypedListTransform = (data: RenderableRecord[]) => RenderableRecord[];

export type TypedListTransform<E extends RenderableRecord<E>> = (data: E[]) => E[];

export interface ListItemProps<E extends RenderableRecord<E>> {
  entry: E;
  index: number;
  style: React.CSSProperties;
  selected: boolean;
  columns: Array<TypedListColumn<E>>;
  onSelect: (key: string) => void;
}
