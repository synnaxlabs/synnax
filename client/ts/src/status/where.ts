// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type label } from "@/label";
import { type Key } from "@/status/types.gen";
import { type Lines } from "@/status/where.gen";

/** @returns the typed lines of a status Where, reading labels through isLabeledBy. */
export const createLines = (
  isLabeledBy: (status: Key, label: label.Key) => boolean,
): Lines => ({
  labels: (status, keys) => keys.some((key) => isLabeledBy(status.key, key)),
});
