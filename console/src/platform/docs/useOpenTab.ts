// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback } from "react";

import { TAB_TYPE } from "@/platform/docs/Docs";
import { Panel } from "@/platform/panel";

export const useOpenTab = (): (() => void) => {
  const openTab = Panel.useOpenTab();
  return useCallback(
    () => openTab({ variant: "view", type: TAB_TYPE, args: {} }, { singleton: true }),
    [openTab],
  );
};
