// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

interface Stringer {
  toString: () => string | number | undefined;
}

export type PureRenderableValue = string | number | undefined;
export type RenderableValue = PureRenderableValue | Stringer;

export const render = (value: RenderableValue): string | number | undefined => {
  if (value === undefined || typeof value === "string" || typeof value === "number")
    return value;
  if (value.toString === undefined) throw new Error("invalid renderer");
  return value.toString();
};
