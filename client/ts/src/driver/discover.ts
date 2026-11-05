// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

/** The encodings of a message payload. */
export const FORMATS = ["json", "schema_registry", "text", "binary"] as const;

export const formatZ = z.enum(FORMATS);

export type Format = z.infer<typeof formatZ>;

/** Decodes the base64 form of a message payload into its bytes. */
export const payloadZ = z
  .string()
  .transform((encoded) => Uint8Array.from(atob(encoded), (c) => c.charCodeAt(0)));
