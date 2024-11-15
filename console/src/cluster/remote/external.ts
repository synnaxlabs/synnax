// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Connect, connectWindowLayout } from "@/cluster/remote/Connect";
import { type Layout } from "@/layout";

export * from "@/cluster/remote/Connect";
export * from "@/cluster/remote/Toolbar";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [connectWindowLayout.type]: Connect,
};
