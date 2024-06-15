// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Create, CREATE_LAYOUT_TYPE } from "@/channel/Create";
import { Layout } from "@/layout";

export * from "@/channel/Create";
export * from "@/channel/link";
export * from "@/channel/ontology";
export * from "@/channel/palette";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [CREATE_LAYOUT_TYPE]: Create,
};
