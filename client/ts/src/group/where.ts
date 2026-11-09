// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Lines } from "@/group/where.gen";
import { resource } from "@/resource";

/** The typed lines of a group Where. */
export const LINES: Lines = {
  parents: ({ parent }, ids) =>
    parent != null && ids.some((id) => resource.idsEqual(id, parent)),
  nested: ({ parent }, nested) => (parent != null) === nested,
};
