// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Layout } from "@/layout";
import {
  MANAGE_CORE_LAYOUT_TYPE,
  ManageCoreModal,
} from "@/manage/ManageCoreModal";
import { COMMANDS as PALETTE_COMMANDS } from "@/manage/palette";
import {
  SNAPSHOTS_LAYOUT_TYPE,
  SnapshotsModal,
} from "@/manage/SnapshotsModal";
import { type Palette } from "@/palette";

export {
  MANAGE_CORE_LAYOUT,
  MANAGE_CORE_LAYOUT_TYPE,
  ManageCoreModal,
} from "@/manage/ManageCoreModal";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [MANAGE_CORE_LAYOUT_TYPE]: ManageCoreModal,
  [SNAPSHOTS_LAYOUT_TYPE]: SnapshotsModal,
};

export const COMMANDS: Palette.Command[] = [...PALETTE_COMMANDS];
