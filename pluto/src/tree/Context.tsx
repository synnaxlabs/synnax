// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import { createContext, type PropsWithChildren, useMemo } from "react";

import { useRequiredContext } from "@/hooks";
import { type Shape } from "@/tree/core";

export interface ContextValue<K extends record.Key = record.Key> {
  shape: Shape<K>;
}

export const Context = createContext<ContextValue | null>(null);

export const useContext = <K extends record.Key>() =>
  useRequiredContext(Context) as ContextValue<K>;

export interface ProviderProps<K extends record.Key>
  extends PropsWithChildren<ContextValue<K>> {}

export const useNodeShape = (index: number) => {
  const { shape } = useContext();
  return shape.nodes[index];
};

export const Provider = <K extends record.Key>({
  shape,
  children,
}: ProviderProps<K>) => {
  const value = useMemo(() => ({ shape }), [shape]);
  return <Context value={value}>{children}</Context>;
};
