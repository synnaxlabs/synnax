// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Where } from "@/panel/types.gen";
import { type project } from "@/project";
import { type user } from "@/user";

/** Matches the shared panels of a project and the drafts that user owns in it. */
export const visibleTo = (proj: project.Key, owner: user.Key): Where => ({
  projects: [proj],
  or: [{ owned: false }, { owners: [owner] }],
});
