// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

const INT8_MIN = -128;
const INT8_MAX = 127;
export const int8 = z.number().int().min(INT8_MIN).max(INT8_MAX);

const INT16_MIN = -32768;
const INT16_MAX = 32767;
export const int16 = z.number().int().min(INT16_MIN).max(INT16_MAX);

const INT32_MIN = -2147483648;
const INT32_MAX = 2147483647;
export const int32 = z.number().int().min(INT32_MIN).max(INT32_MAX);

const INT64_MIN = -9223372036854775808n;
const INT64_MAX = 9223372036854775807n;
export const int64 = z.bigint().min(INT64_MIN).max(INT64_MAX);

const UINT8_MAX = 255;
export const uint8 = z.number().int().min(0).max(UINT8_MAX);

const UINT16_MAX = 65535;
export const uint16 = z.number().int().min(0).max(UINT16_MAX);

const UINT32_MAX = 4294967295;
export const uint32 = z.number().int().min(0).max(UINT32_MAX);

const UINT64_MAX = 18446744073709551615n;
export const uint64 = z.bigint().min(0n).max(UINT64_MAX);
