import { type record } from "@synnaxlabs/x";
import { useState } from "react";

import { Dialog as CoreDialog } from "@/dialog";
import { Input } from "@/input";
import { List } from "@/list";
import { type state } from "@/state";

interface SearchParams {
  term?: string;
}

export interface SearchInputProps<P extends SearchParams> {
  searchPlaceholder?: string;
  onSearch: state.Setter<P, P | {}>;
}

export const SearchInput = <P extends SearchParams>({
  searchPlaceholder = "Search...",
  onSearch,
}: SearchInputProps<P>) => {
  const [term, setTerm] = useState<string>("");
  return (
    <Input.Text
      value={term}
      placeholder={searchPlaceholder}
      borderShade={6}
      onChange={(v) => {
        setTerm(v);
        onSearch((prev) => ({ ...prev, term: v }) as P);
      }}
    />
  );
};

export interface DialogProps<K extends record.Key, P extends SearchParams>
  extends Omit<CoreDialog.DialogProps, "children">,
    SearchInputProps<P>,
    Pick<List.ItemsProps<K>, "emptyContent" | "children"> {
  onSearch: state.Setter<P, P | {}>;
}

export const Dialog = <K extends record.Key, P extends SearchParams>({
  onSearch,
  children,
  emptyContent,
  searchPlaceholder,
  ...rest
}: DialogProps<K, P>) => (
  <CoreDialog.Dialog {...rest}>
    <SearchInput onSearch={onSearch} searchPlaceholder={searchPlaceholder} />
    <List.Items emptyContent={emptyContent} bordered borderShade={6}>
      {children}
    </List.Items>
  </CoreDialog.Dialog>
);
