// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useEffect, useState } from "react";

import { KeyedRenderableRecord } from "@synnaxlabs/x";

import { useListContext } from "./ListContext";

import { Input as DefaultInput, InputControl } from "@/core/Input";
import { useSearchTransform, UseSearchTransformProps } from "@/hooks";
import { RenderProp } from "@/util/renderProp";

export interface ListSearchProps<E extends KeyedRenderableRecord<E>>
  extends Omit<UseSearchTransformProps<E>, "query"> {
  children?: RenderProp<InputControl<string>>;
}

export const ListSearch = <E extends KeyedRenderableRecord<E>>({
  children = (props) => <DefaultInput {...props} />,
  opts,
}: ListSearchProps<E>): JSX.Element | null => {
  const [value, setValue] = useState("");

  const search = useSearchTransform<E>({ query: value, opts });

  const { setTransform } = useListContext<E>();

  useEffect(() => setTransform("search", search), [search]);
  const onChange = useCallback((v: any) => setValue(v), []);

  return children({ value, onChange });
};
