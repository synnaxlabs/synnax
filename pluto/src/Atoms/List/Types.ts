export type Key = string | number;

export type CoreColumn = {
  label: string;
  visible?: boolean;
  width?: number;
};

export type UntypedColumn = {
  key: string;
} & CoreColumn;

export type TypedColumn<K extends Key, E extends TypedListEntry<K>> = {
  key: keyof E;
} & CoreColumn;

export type CoreListEntry = {
  [key: string]: any;
};

export type UntypedListEntry = {
  key: string;
} & CoreListEntry;

export type TypedListEntry<K extends Key> = {
  key: K;
} & CoreListEntry;

export type UntypedTransform = (data: UntypedListEntry[]) => UntypedListEntry[];

export type TypedTransform<K extends Key, E extends TypedListEntry<K>> = (
  data: E[]
) => E[];

export type ListItemProps<K extends Key, E extends TypedListEntry<K>> = {
  entry: E;
  index: number;
  style: React.CSSProperties;
  selected: boolean;
  columns: TypedColumn<K, E>[];
  onSelect: (key: K) => void;
};
