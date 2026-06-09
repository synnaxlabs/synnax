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
import { useDispatch, useStore } from "react-redux";

import { Layout } from "@/layout";
import { Project } from "@/project";
import { useSelectPendingUpload } from "@/schematic/selectors";
import { clearPendingUpload } from "@/schematic/slice";
import { type RootState } from "@/store";

export const useAutoUpload = (key: string): boolean => {
  const store = useStore<RootState>();
  const pendingUpload = useSelectPendingUpload(key);
  const projectKey = Project.useSelectActiveKey();
  const dispatch = useDispatch();
  const { update: create } = Schematic.useCreate({
    afterSuccess: useCallback(
      ({ data: { key } }: Flux.AfterSuccessParams<Schematic.UseCreateResult>) => {
        dispatch(clearPendingUpload({ key }));
      },
      [dispatch],
    ),
  });
  useEffect(() => {
    if (pendingUpload == null) return;
    const name = Layout.selectRequiredName(store.getState(), key);
    create({ ...pendingUpload, project: projectKey ?? undefined, name });
  }, [pendingUpload, projectKey, key, create]);
  return pendingUpload == null;
};
