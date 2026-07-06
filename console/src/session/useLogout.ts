// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback } from "react";

import { Layout } from "@/session/layout";
import { Nav } from "@/session/nav";
import { Node } from "@/session/node";
import { Project } from "@/session/project";
import { useDispatch } from "@/session/store";

export const useLogout = () => {
  const dispatch = useDispatch();
  return useCallback(() => {
    dispatch(Node.clearSelected());
    dispatch(Project.clearSelected());
    dispatch(Layout.clearProject());
    dispatch(Nav.hideAll({}));
  }, [dispatch]);
};
