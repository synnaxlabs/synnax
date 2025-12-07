// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Synnax } from "@synnaxlabs/pluto";
import { useEffect } from "react";
import { useDispatch } from "react-redux";

import { useSelectActiveKey } from "@/cluster/selectors";
import { changeKey } from "@/cluster/slice";

// useSyncClusterKey synchronizes the actual cluster key of the cluster to the cluster
// key in the redux store. This is needed for a few different reasons, such as
// connecting to a different cluster at the same address or connecting to the local or
// demo cluster that is already defined in the initial slice state.
export const useSyncClusterKey = () => {
  const activeClusterKey = useSelectActiveKey();
  const { clusterKey, status } = Synnax.useConnectionState();
  const dispatch = useDispatch();
  useEffect(() => {
    if (
      status !== "connected" ||
      activeClusterKey == null ||
      activeClusterKey === clusterKey
    )
      return;
    dispatch(changeKey({ oldKey: activeClusterKey, newKey: clusterKey }));
  }, [status]);
};
