// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deepDelete } from "@/deep/delete";
import { deepEqual, deepPartialEqual } from "@/deep/equal";
import { deepMerge } from "@/deep/merge";
export type { DeepKey } from "@/deep/key";
export type { DeepPartial } from "@/deep/partial";

export const Deep = {
  merge: deepMerge,
  delete: deepDelete,
  equal: deepEqual,
  partialEqual: deepPartialEqual,
};
