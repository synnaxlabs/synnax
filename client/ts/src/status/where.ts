// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { resource } from "@/resource";
import { type Lines } from "@/status/where.gen";

/** The typed lines of a status Where. */
export const LINES: Lines = {
  labels: (status, keys) => status.labels.some((key) => keys.includes(key)),
  owners: ({ owner }, owners) =>
    owner != null && owners.some((o) => resource.idsEqual(o, owner)),
};
