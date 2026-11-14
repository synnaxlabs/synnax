// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger } from "@synnaxlabs/client";

import { Panel } from "@/platform/panel";

/** @returns a function that opens the overview of a range in a tab. */
export const useOpen = (): ((key: ranger.Key) => void) => {
  const openTab = Panel.useOpenTab();
  return (key) => openTab({ variant: "resource", resource: ranger.resourceID(key) });
};
