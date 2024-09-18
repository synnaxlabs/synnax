// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Layout } from "@/layout";
import { Overview, OVERVIEW_LAYOUT_TYPE } from "@/user/Overview";
import { REGISTER_LAYOUT_TYPE, RegisterModal } from "@/user/RegisterModal";

export * from "@/user/Overview";
export * from "@/user/RegisterModal";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [REGISTER_LAYOUT_TYPE]: RegisterModal,
  [OVERVIEW_LAYOUT_TYPE]: Overview,
};
