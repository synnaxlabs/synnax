// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useEffect, useState } from "react";

import { KeyedRenderableRecord } from "@synnaxlabs/x";

import { Input as DefaultInput, InputControl } from "@/core/Input";
import { useListContext } from "@/core/List/ListContext";
import { useSearchTransform, UseSearchTransformProps } from "@/hooks";
import { useDebouncedCallback } from "@/util/debounce";
import { RenderProp } from "@/util/renderProp";

export interface ListSearchProps<E extends KeyedRenderableRecord<E>>
  extends Omit<UseSearchTransformProps<E>, "query"> {
  children?: RenderProp<InputControl<string>>;
  debounce?: number;
}

export const ListSearch = <E extends KeyedRenderableRecord<E>>({
  children = (props) => <DefaultInput {...props} />,
  debounce = 250,
  opts,
}: ListSearchProps<E>): ReactElement | null => {
  const [value, setValue] = useState("");

  const search = useSearchTransform<E>({ query: value, opts });

  const { setTransform } = useListContext<E>();

  useEffect(() => setTransform("search", search), [search]);
  const onChange = useDebouncedCallback((v: any) => setValue(v), debounce, [setValue]);

  return children({ value, onChange });
};
