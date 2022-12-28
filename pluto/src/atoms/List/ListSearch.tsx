import { useCallback, useEffect, useState } from "react";

import { useListContext } from "./ListContext";

import { Input as DefaultInput, InputControlProps, InputProps } from "@/atoms/Input";
import { useSearch, UseSearchProps } from "@/hooks";
import { RenderableRecord } from "@/util/record";

export interface ListSearchProps<E extends RenderableRecord<E>>
  extends Omit<InputProps, "children" | "onChange" | "value">,
    Omit<UseSearchProps<E>, "query"> {
  children?: (props: InputControlProps<string>) => JSX.Element;
  debounce?: number;
}

export const ListSearch = <E extends RenderableRecord<E>>({
  children = (props) => <DefaultInput {...props} />,
  opts,
}: ListSearchProps<E>): JSX.Element => {
  const [value, setValue] = useState("");

  const search = useSearch<E>({ query: value, opts });
  const { setTransform } = useListContext<E>();
  useEffect(() => setTransform("search", search), [search]);

  const onChange = useCallback((v: any) => setValue(v), []);

  return children({ value, onChange });
};
