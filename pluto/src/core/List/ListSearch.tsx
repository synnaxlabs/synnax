// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useEffect, useState } from "react";

import { RenderableRecord } from "@synnaxlabs/x";

import { useListContext } from "./ListContext";

import { Input as DefaultInput, InputControlProps, InputProps } from "@/core/Input";
import { useSearch, UseSearchProps } from "@/hooks";

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
