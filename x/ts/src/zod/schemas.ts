// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { math } from "@/math";

// Signed integers
export const int8Z = z.number().int().min(math.MIN_INT8).max(math.MAX_INT8);
export const int16Z = z.number().int().min(math.MIN_INT16).max(math.MAX_INT16);
export const int32Z = z.number().int().min(math.MIN_INT32).max(math.MAX_INT32);
export const int64Z = z
  .number()
  .int()
  .min(math.MIN_INT64_NUMBER)
  .max(math.MAX_INT64_NUMBER);

// Unsigned integers
export const uint8Z = z.number().int().min(0).max(math.MAX_UINT8);
export const uint12Z = z.number().int().min(0).max(math.MAX_UINT12);
export const uint16Z = z.number().int().min(0).max(math.MAX_UINT16);
export const uint32Z = z.number().int().min(0).max(math.MAX_UINT32);
export const uint64Z = z.number().int().min(0).max(math.MAX_UINT64_NUMBER);

// Floats
export const float32Z = z.number();
export const float64Z = z.number();

/** @deprecated Use uint12Z instead */
export const uint12 = uint12Z;
