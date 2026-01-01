// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Button } from "@/button";
import { type Dialog } from "@/dialog";

export type Variant = Dialog.FrameProps["variant"] | "preview";

export const transformDialogVariant = (
  variant: Variant,
): Dialog.FrameProps["variant"] => (variant === "preview" ? "connected" : variant);

export const transformTriggerVariant = (
  variant: Variant,
): Button.ButtonProps["variant"] => (variant === "preview" ? "preview" : undefined);
