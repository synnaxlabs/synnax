// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Flux, Schematic } from "@synnaxlabs/pluto";
import { useCallback, useEffect } from "react";
import { useDispatch } from "react-redux";

import { Layout } from "@/layout";
import { useSelectPendingUpload } from "@/schematic/selectors";
import { clearPendingUpload } from "@/schematic/slice";
import { Workspace } from "@/workspace";

export const useAutoUpload = (key: string): boolean => {
  const pendingUpload = useSelectPendingUpload(key);
  const name = Layout.useSelectRequiredName(key);
  const workspaceKey = Workspace.useSelectActiveKey();
  const dispatch = useDispatch();
  const { update: create } = Schematic.useCreate({
    afterSuccess: useCallback(
      ({ data: { key } }: Flux.AfterSuccessParams<Schematic.UseCreateResult>) => {
        // eslint-disable-next-line no-console
        console.log("[useAutoUpload] afterSuccess clearing pending for key=", key);
        dispatch(clearPendingUpload({ key }));
      },
      [dispatch],
    ),
  });
  // eslint-disable-next-line no-console
  console.log(
    "[useAutoUpload] key=",
    key,
    "pendingUpload=",
    pendingUpload != null,
    "pendingKey=",
    pendingUpload?.key,
    "workspaceKey=",
    workspaceKey,
    "name=",
    name,
  );
  useEffect(() => {
    if (pendingUpload == null) return;
    // eslint-disable-next-line no-console
    console.log(
      "[useAutoUpload] firing create for key=",
      key,
      "pendingKey=",
      pendingUpload.key,
    );
    create({ ...pendingUpload, workspace: workspaceKey ?? undefined, name });
  }, [pendingUpload, workspaceKey, key, create, name]);
  return pendingUpload == null;
};
