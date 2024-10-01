// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

export const actionZ = z.union([
  z.literal("all"),
  z.literal("create"),
  z.literal("delete"),
  z.literal("retrieve"),
  z.literal("update"),
]);
export type Action = z.infer<typeof actionZ>;

export const ALL_ACTION: Action = "all";
export const CREATE_ACTION: Action = "create";
export const DELETE_ACTION: Action = "delete";
export const RETRIEVE_ACTION: Action = "retrieve";
export const UPDATE_ACTION: Action = "update";
