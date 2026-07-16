// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { cache } from "@synnaxlabs/client";

export type FetchOptions = cache.FetchOptions;
export type Query = cache.Query;
export type Data = cache.Data;
export type Verbs = cache.Verbs;

export const RENAME_VERBS = cache.RENAME_VERBS;
export const DELETE_VERBS = cache.DELETE_VERBS;
export const UPDATE_VERBS = cache.UPDATE_VERBS;
export const CREATE_VERBS = cache.CREATE_VERBS;
export const SNAPSHOT_VERBS = cache.SNAPSHOT_VERBS;
export const COPY_VERBS = cache.COPY_VERBS;
export const SET_VERBS = cache.SET_VERBS;
export const SAVE_VERBS = cache.SAVE_VERBS;
