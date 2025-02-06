// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Stringer } from "@/primitive";

export type PureRenderableValue = string | number | undefined;
export type RenderableValue = PureRenderableValue | Stringer;

export const convertRenderV = (value: RenderableValue): string | number | undefined => {
  if (value === undefined || typeof value === "string" || typeof value === "number")
    return value;
  if (value.toString === undefined) throw new Error("invalid renderer");
  return value.toString();
};
