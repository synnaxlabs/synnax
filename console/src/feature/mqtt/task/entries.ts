// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createLeaves, createRawLeaf, type Leaf } from "@/feature/mqtt/task/leaves";
import { type BrowsedTopic } from "@/feature/mqtt/task/types";

const unescapeToken = (token: string): string =>
  token.replaceAll("~1", "/").replaceAll("~0", "~");

const tokens = (pointer: string): string[] =>
  pointer === "" ? [] : pointer.slice(1).split("/").map(unescapeToken);

/** Names a pointer for people: "motor › speed", or "Whole payload". */
export const pointerLabel = (pointer: string): string =>
  pointer === "" ? "Whole payload" : tokens(pointer).join(" › ");

/** @returns the sample of a browsed topic at a pointer, or null when it has none. */
export const sampleAt = (
  { payload, fields }: BrowsedTopic,
  pointer: string,
): string | null => {
  const field = fields.find((f) => f.pointer === pointer);
  if (field != null) return field.sample;
  return pointer === "" ? payload : null;
};

/**
 * The values a browsed topic offers. A payload that is not JSON offers itself as one
 * string.
 */
export const leavesOf = ({ payload, fields }: BrowsedTopic): Leaf[] => {
  const leaves = createLeaves(fields);
  if (leaves.length > 0 || payload === "") return leaves;
  return [createRawLeaf()];
};
