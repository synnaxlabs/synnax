// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel, project } from "@synnaxlabs/client";
import { type Flux, Panel } from "@synnaxlabs/pluto";
import { useCallback } from "react";
import { useDispatch } from "react-redux";

import { Session } from "@/session";

export const useCreate = (): (() => void) => {
  const dispatch = useDispatch();
  const getProjectKey = Session.Project.useGetSelected();
  const { update: create } = Panel.useCreate({
    afterOptimistic: useCallback(
      ({ data: { key } }: Flux.AfterOptimisticParams<panel.Panel>) => {
        dispatch(Session.Panel.select({ key }));
      },
      [dispatch],
    ),
  });
  return useCallback(() => {
    create({ name: "New Panel", parent: project.ontologyID(getProjectKey()) });
  }, [create, getProjectKey]);
};
