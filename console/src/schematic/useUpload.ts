// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { Schematic, Status } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { useEffect, useRef } from "react";
import { useDispatch } from "react-redux";

import { useSelectPendingUpload } from "@/schematic/selectors";
import { clearPendingUpload } from "@/schematic/slice";
import { Workspace } from "@/workspace";

// useAutoUpload pushes a v5→v6 migrated schematic's local graph state to the
// server on first mount. The v6 console migration parks v0–v5 graph fields in
// pendingUpload because the canvas now reads from Pluto / the server. This
// hook is the bridge that ensures the schematic exists on the server before
// the canvas tries to load it.
//
// Schematics that came from a workspace already exist on the server and have
// no pendingUpload. Schematics that were standalone (no workspace) only lived
// in console redux state pre-v6 — those are the ones this hook lifts to the
// server, attached to the user's currently active workspace if any.
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
        const payload = {
          key,
          name,
          snapshot: pendingUpload.snapshot,
          authority: pendingUpload.authority ?? 1,
          legend:
            (pendingUpload.legend as schematic.Legend | undefined) ?? {
              visible: false,
              position: { x: 50, y: 50, units: { x: "px", y: "px" } },
              colors: {},
            },
          nodes: pendingUpload.nodes as schematic.Node[],
          edges: pendingUpload.edges as schematic.Edge[],
          props: pendingUpload.props as Record<string, record.Unknown>,
          workspace: workspaceKey ?? undefined,
        };
        create(payload);
        dispatch(clearPendingUpload({ key }));
      } finally {
        inFlight.current = false;
      }
    }, `Failed to migrate schematic ${name} to the server`);
  }, [pendingUpload, workspaceKey, key, name, dispatch, create, handleError]);
};
