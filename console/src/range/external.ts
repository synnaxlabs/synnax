// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Layout } from "@/layout";
import { Edit, EDIT_LAYOUT_TYPE } from "@/range/EditLayout";
import { MetaData, metaDataWindowLayout } from "@/range/MetaData";

export * from "@/range/ContextMenu";
export * from "@/range/EditLayout";
export * from "@/range/link";
export * from "@/range/MetaData";
export * from "@/range/ontology";
export * from "@/range/palette";
export * from "@/range/range";
export * from "@/range/Select";
export * from "@/range/selectors";
export * from "@/range/slice";
export * from "@/range/Toolbar";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [EDIT_LAYOUT_TYPE]: Edit,
  [metaDataWindowLayout.type]: MetaData,
};
