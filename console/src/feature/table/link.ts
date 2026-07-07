// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { table } from "@synnaxlabs/client";
import { useCallback } from "react";

import { type Link } from "@/platform/link";
import { Panel } from "@/platform/panel";

export const useLink = (): Link.Handler => {
  const openTab = Panel.useOpenTab();
  return useCallback(
    async ({ client, key }) => {
      const t = await client.tables.retrieve({ key });
      openTab({ variant: "resource", resource: table.ontologyID(t.key) });
    },
    [openTab],
  );
};

export const LINKS: Link.Registry = { table: useLink };
