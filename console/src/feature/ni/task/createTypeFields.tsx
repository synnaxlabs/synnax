// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep } from "@synnaxlabs/x";
import { type FC } from "react";

interface TypeFormProps<V extends { type: string }, K extends V["type"]> {
  path: deep.Path<Extract<V, { type: K }>>;
}

interface TypeFieldsProps<V extends { type: string }, K extends V["type"]> {
  type: K;
  path: deep.Path<V>;
}

/**
 * Builds a component that renders the form for a value's type.
 *
 * @param forms One form per type. Each form gets a path narrowed to its type.
 */
export const createTypeFields = <V extends { type: string }>(forms: {
  [K in V["type"]]: FC<TypeFormProps<V, K>>;
}) => {
  const TypeFields = <K extends V["type"]>({ type, path }: TypeFieldsProps<V, K>) => {
    const TypeForm: FC<TypeFormProps<V, K>> = forms[type];
    return <TypeForm path={deep.variant(path, "type", type)} />;
  };
  return TypeFields;
};
