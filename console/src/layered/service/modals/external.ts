// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export * from "@/layered/service/modals/Body";
export {
  type ConfirmParams,
  type PromptConfirm,
  useConfirm,
} from "@/layered/service/modals/Confirm";
export {
  create,
  type Opener,
  type OpenHook,
  type Prompt,
  createPrompt as prompt,
  type PromptHook,
} from "@/layered/service/modals/factory";
export * from "@/layered/service/modals/Footer";
export { Header } from "@/layered/service/modals/Header";
export {
  type PromptRename,
  type RenameParams,
  useRename,
} from "@/layered/service/modals/Rename";
export { type Content, type ContentProps } from "@/layered/session/modals/store";
