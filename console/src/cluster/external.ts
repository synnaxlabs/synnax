// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

import { Connect, connectWindowLayout } from "@/cluster/Connect";
import { Layout } from "@/layout";

export * from "@/cluster/Badges";
export * from "@/cluster/Connect";
export * from "@/cluster/core";
export * from "@/cluster/Dropdown";
export * from "@/cluster/selectors";
export * from "@/cluster/slice";
export * from "@/cluster/testConnection";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [connectWindowLayout.type]: Connect,
};
