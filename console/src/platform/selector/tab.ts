// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel } from "@synnaxlabs/client";
import { useCallback } from "react";

import { Panel } from "@/platform/panel";

export const TAB_TYPE = "selector";

export const createEmptyTab = (): panel.NewTab => ({
  variant: "view",
  type: TAB_TYPE,
  args: {},
});

export const useOpenTab = (): (() => void) => {
  const openTab = Panel.useOpenTab();
  return useCallback(() => openTab(createEmptyTab()), [openTab]);
};
