// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Layout } from "@/layout";
import { SET_LAYOUT_TYPE, SetModal } from "@/permissions/SetLayout";

export * from "@/permissions/selectors";
export * from "@/permissions/SetLayout";
export * from "@/permissions/slice";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [SET_LAYOUT_TYPE]: SetModal,
};
