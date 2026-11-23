// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd } from "@synnaxlabs/client";
import type { json } from "@synnaxlabs/x";

/** @returns a field named name that reads the values of an inferred field. */
export const createField = (
  {
    pointer,
    kind,
    timeFormat,
  }: Pick<json.InferredField, "pointer" | "kind" | "timeFormat">,
  name: string,
): icd.JSONField => icd.jsonFieldZ.parse({ name, pointer, type: kind, timeFormat });
