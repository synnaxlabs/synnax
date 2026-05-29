// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Flux, Table as PTable } from "@synnaxlabs/pluto";
import { useCallback, useEffect } from "react";
import { useDispatch } from "react-redux";

import { Layout } from "@/layout";
import { useSelectPendingUpload } from "@/table/selectors";
import { clearPendingUpload } from "@/table/slice";
import { Workspace } from "@/workspace";

export const useAutoUpload = (key: string): boolean => {
  const pendingUpload = useSelectPendingUpload(key);
  const name = Layout.useSelectRequiredName(key);
  const workspaceKey = Workspace.useSelectActiveKey();
  const dispatch = useDispatch();
  const { update: create } = PTable.useCreate({
    afterSuccess: useCallback(
      ({ data: { key } }: Flux.AfterSuccessParams<PTable.CreateOutput>) => {
        dispatch(clearPendingUpload({ key }));
      },
      [dispatch],
    ),
  });
  useEffect(() => {
    if (pendingUpload == null) return;
    create({ ...pendingUpload, workspace: workspaceKey ?? undefined, name });
  }, [pendingUpload, workspaceKey, key, create, name]);
  return pendingUpload == null;
};
