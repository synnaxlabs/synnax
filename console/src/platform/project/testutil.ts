// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type project } from "@synnaxlabs/client";

import { Session } from "@/session";

/** A project slice with the given project set as the active (selected) project. */
export const createActiveState = (
  proj: project.Project,
): Session.Project.SliceState => ({
  ...Session.Project.ZERO_SLICE_STATE,
  selected: proj.key,
});
