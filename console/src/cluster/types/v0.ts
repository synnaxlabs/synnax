// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type SynnaxProps } from "@synnaxlabs/client";

export const VERSION = "0.0.1";
export type Version = typeof VERSION;

export type Cluster = { key: string; name: string; props: SynnaxProps };

export type SliceState = {
  version: Version;
  activeCluster: string | null;
  clusters: Record<string, Cluster>;
};

export const ZERO_SLICE_STATE: SliceState = {
  version: VERSION,
  activeCluster: null,
  clusters: {},
};
