export type Key = string | number;

type CoreListColumn = {
  label: string;
  visible?: boolean;
  width?: number;
};

export type UntypedListColumn = {
  key: string;
} & CoreListColumn;

export type TypedListColumn<K extends Key, E extends TypedListEntry<K>> = {
  key: keyof E;
} & CoreListColumn;

type CoreListEntry = {
  [key: string]: any;
};

export type UntypedListEntry = {
  key: string;
} & CoreListEntry;

export type TypedListEntry<K extends Key> = {
  key: K;
} & CoreListEntry;

export type UntypedListTransform = (
  data: UntypedListEntry[]
) => UntypedListEntry[];

export type TypedListTransform<K extends Key, E extends TypedListEntry<K>> = (
  data: E[]
) => E[];

export type ListItemProps<K extends Key, E extends TypedListEntry<K>> = {
  entry: E;
  index: number;
  style: React.CSSProperties;
  selected: boolean;
  columns: TypedListColumn<K, E>[];
  onSelect: (key: K) => void;
};
