// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { framer } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";

/**
 * Reports whether the subject may command hardware. Starting, stopping, and deploying a
 * task, and taking control of a schematic, all write a command frame and never the
 * resource itself, so they answer to framer create rather than an update on the task or
 * the schematic.
 */
export const useCanCommand = (): boolean =>
  Access.useCreateGranted(framer.TYPE_ONTOLOGY_ID);
