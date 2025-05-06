// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { stringify, v4 } from "uuid";
import z from "zod";

export const create = (): string => v4();

export const uuidZ = z.union([
  z.uuid(),
  z.instanceof(Uint8Array).transform((bytes) => stringify(bytes)),
]);

export type UUID = z.infer<typeof uuidZ>;
