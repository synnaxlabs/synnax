// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback } from "react";

import { Cluster } from "@/session/cluster";
import { Nav } from "@/session/nav";
import { Panel } from "@/session/panel";
import { Project } from "@/session/project";
import { useDispatch } from "@/session/store";

export const useLogout = () => {
  const dispatch = useDispatch();
  return useCallback(() => {
    // The cluster clear must come last: it flushes the cluster's persistence
    // partition, which must already hold the cleared project selection.
    dispatch(Project.clearSelected());
    dispatch(Panel.reset());
    dispatch(Nav.hideAll({}));
    dispatch(Cluster.clearSelected());
  }, [dispatch]);
};
