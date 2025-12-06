// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

export const ALL_ACTION = "all";
export const CREATE_ACTION = "create";
export const DELETE_ACTION = "delete";
export const RETRIEVE_ACTION = "retrieve";
export const UPDATE_ACTION = "update";

export const actionZ = z.enum([
  ALL_ACTION,
  CREATE_ACTION,
  DELETE_ACTION,
  RETRIEVE_ACTION,
  UPDATE_ACTION,
]);
export type Action = z.infer<typeof actionZ>;
