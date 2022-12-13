type ListColumn = {
  label: string;
  visible?: boolean;
  width?: number;
};

export type UntypedListColumn = {
  key: string;
} & ListColumn;

export type TypedListColumn<E extends ListEntry> = {
  key: keyof E;
} & ListColumn;

type CoreListEntry = {
  [key: string]: string | number;
};

export type ListEntry = {
  key: string;
} & CoreListEntry;

export type UntypedListTransform = (data: ListEntry[]) => ListEntry[];

export type TypedListTransform<E extends ListEntry> = (data: E[]) => E[];

export type ListItemProps<E extends ListEntry> = {
  entry: E;
  index: number;
  style: React.CSSProperties;
  selected: boolean;
  columns: TypedListColumn<E>[];
  onSelect: (key: string) => void;
};
