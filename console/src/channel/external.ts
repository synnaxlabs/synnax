// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CREATE_LAYOUT_TYPE, CreateModal } from "@/channel/Create";
import { Layout } from "@/layout";

export * from "@/channel/Create";
export * from "@/channel/services/ontology";
export * from "@/channel/services/palette";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [CREATE_LAYOUT_TYPE]: CreateModal,
};
