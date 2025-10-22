// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Synnax } from "@synnaxlabs/client";

import { Cluster } from "@/cluster";
import { type Link } from "@/link";

export const handleLink: Link.ClusterHandler = async ({ store, key }) => {
  const cluster = Cluster.select(store.getState(), key);
  if (cluster == null) throw new Error(`Core with key ${key} not found`);
  store.dispatch(Cluster.setActive(key));
  return new Synnax(cluster);
};
