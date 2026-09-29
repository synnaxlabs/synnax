// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { library } from "@synnaxlabs/client";
import { Library } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Panel } from "@/platform/panel";

/** @returns a function that creates an empty library and opens it in a tab. */
export const useCreate = (): (() => void) => {
  const openTab = Panel.useOpenTab();
  const { update } = Library.useCreate({
    afterOptimistic: ({ data: { key } }) =>
      openTab({ variant: "resource", resource: library.ontologyID(key) }),
  });
  return useCallback(() => update({ name: "Library", entries: [] }), [update]);
};
