// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Layout } from "@/layout";
import { Confirm, CONFIRM_LAYOUT_TYPE } from "@/modals/Confirm";
import { Rename, RENAME_LAYOUT_TYPE } from "@/modals/Rename";

export * from "@/modals/Base";
export * from "@/modals/BottomNavBar";
export {
  type PromptConfirm,
  type PromptConfirmLayoutArgs,
  useConfirm,
} from "@/modals/Confirm";
export * from "@/modals/layout";
export {
  type PromptRename,
  type PromptRenameLayoutArgs,
  useRename,
} from "@/modals/Rename";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [CONFIRM_LAYOUT_TYPE]: Confirm,
  [RENAME_LAYOUT_TYPE]: Rename,
};
