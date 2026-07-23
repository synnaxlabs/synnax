// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback } from "react";

import { retrieveAndOpenTab } from "@/feature/task/tabs";
import { type Link } from "@/platform/link";
import { Panel } from "@/platform/panel";

export const useLink = (): Link.Handler => {
  const openTab = Panel.useOpenTab();
  return useCallback(
    async ({ client, key }) => await retrieveAndOpenTab(client, key, openTab),
    [openTab],
  );
};

export const LINKS: Link.Registry = { task: useLink };
