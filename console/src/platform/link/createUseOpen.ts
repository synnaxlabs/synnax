// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type resource } from "@synnaxlabs/client";

import { type UseHandler } from "@/platform/link/types";
import { Panel } from "@/platform/panel";

/** Creates a handler that opens the linked resource of the given type in a tab. */
// Marked pure, so a bundler drops a handler that nothing uses.
/* #__NO_SIDE_EFFECTS__ */
export const createUseOpenResourceTab = (type: resource.Type): UseHandler => {
  const useOpenResourceTab: UseHandler = () => {
    const openTab = Panel.useOpenTab();
    return async ({ key }) => openTab({ variant: "resource", resource: { type, key } });
  };
  return useOpenResourceTab;
};
