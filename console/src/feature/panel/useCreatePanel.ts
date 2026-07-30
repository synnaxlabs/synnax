// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { project } from "@synnaxlabs/client";
import { Panel } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { useCallback } from "react";
import { useDispatch } from "react-redux";

import { Session } from "@/session";

/** Returns a callback that creates a panel in the selected project and selects it. */
export const useCreatePanel = (): (() => void) => {
  const dispatch = useDispatch();
  const projectKey = Session.Project.useSelectSelected();
  const { update: create } = Panel.useCreate();
  return useCallback(() => {
    const key = uuid.create();
    create({ key, name: "New Panel", parent: project.ontologyID(projectKey) });
    dispatch(Session.Panel.select({ key }));
  }, [create, dispatch, projectKey]);
};
