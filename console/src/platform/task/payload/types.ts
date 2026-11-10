// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type json } from "@synnaxlabs/x";

/** The encoding of a timestamp carried as a JSON value. */
export type TimeFormat = "iso8601" | "unix_sec" | "unix_ms" | "unix_us" | "unix_ns";

/** The source of a value that each write generates. */
export type GeneratorType = "uuid" | "timestamp";

/** The JSON type that a written value is serialized as. */
export type JSONType = Exclude<json.PrimitiveType, "null">;
