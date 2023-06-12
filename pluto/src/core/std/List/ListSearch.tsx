// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useEffect, useState } from "react";

import { Key, KeyedRenderableRecord } from "@synnaxlabs/x";

import { useSearchTransform, UseSearchTransformProps } from "@/core/hooks";
import { Input as DefaultInput, InputControl } from "@/core/std/Input";
import { useListContext } from "@/core/std/List/ListContext";
import { useDebouncedCallback } from "@/util/debounce";
import { RenderProp } from "@/util/renderProp";

export interface ListSearchProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
> extends Omit<UseSearchTransformProps<E>, "query"> {
  children?: RenderProp<InputControl<string>>;
  debounce?: number;
}

export const ListSearch = <
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>
>({
  children = (props) => <DefaultInput {...props} />,
  debounce = 250,
  opts,
}: ListSearchProps<K, E>): ReactElement | null => {
  const [value, setValue] = useState("");

  const search = useSearchTransform<E>({ query: value, opts });

  const { setTransform } = useListContext<K, E>();

  useEffect(() => setTransform("search", search), [search]);
  const onChange = useDebouncedCallback((v: any) => setValue(v), debounce, [setValue]);

  return children({ value, onChange });
};
