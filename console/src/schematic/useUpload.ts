// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Schematic, Status } from "@synnaxlabs/pluto";
import { useEffect, useRef } from "react";
import { useDispatch } from "react-redux";

import { useSelectPendingUpload } from "@/schematic/selectors";
import { clearPendingUpload } from "@/schematic/slice";
import { Workspace } from "@/workspace";

// useAutoUpload uploads a schematic's pendingUpload to the server and clears
// it. Used to lift schematics migrated from pre-v6 console state, which only
// existed locally, onto the server.
export const useAutoUpload = (key: string, name: string): void => {
  const pendingUpload = useSelectPendingUpload(key);
  const workspaceKey = Workspace.useSelectActiveKey();
  const { update: create } = Schematic.useCreate();
  const dispatch = useDispatch();
  const handleError = Status.useErrorHandler();
  const inFlight = useRef(false);

  useEffect(() => {
    if (pendingUpload == null || inFlight.current) return;
    inFlight.current = true;
    handleError(async () => {
      try {
        create({
          key,
          name,
          snapshot: pendingUpload.snapshot,
          authority: pendingUpload.authority ?? 1,
          legend: pendingUpload.legend,
          nodes: pendingUpload.nodes,
          edges: pendingUpload.edges,
          configs: pendingUpload.configs,
          workspace: workspaceKey ?? undefined,
        });
        dispatch(clearPendingUpload({ key }));
      } finally {
        inFlight.current = false;
      }
    }, `Failed to migrate schematic ${name} to the server`);
  }, [pendingUpload, workspaceKey, key, name, dispatch, create, handleError]);
};
