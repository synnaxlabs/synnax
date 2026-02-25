// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { math } from "@/math";

export const int8 = z.int().min(math.MIN_INT8).max(math.MAX_INT8);
export const int16 = z.int().min(math.MIN_INT16).max(math.MAX_INT16);
export const int64 = z.int().min(math.MIN_INT64_NUMBER).max(math.MAX_INT64_NUMBER);

export const uint8 = z.int().min(0).max(math.MAX_UINT8);
export const uint12 = z.int().min(0).max(math.MAX_UINT12);
export const uint16 = z.int().min(0).max(math.MAX_UINT16);
export const uint20 = z.int().min(0).max(math.MAX_UINT20);
