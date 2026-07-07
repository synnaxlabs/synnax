// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger } from "@synnaxlabs/client";
import { useCallback } from "react";

import { type Link } from "@/platform/link";
import { Panel } from "@/platform/panel";
import { Session } from "@/session";

export const useLink = (): Link.Handler => {
  const dispatch = Session.useDispatch();
  const openTab = Panel.useOpenTab();
  return useCallback(
    async ({ client, key }) => {
      const range = await client.ranges.retrieve(key);
      Session.Range.fromClient(range).forEach((r) => dispatch(Session.Range.add(r)));
      dispatch(Session.Range.select(range.key));
      openTab({ variant: "resource", resource: ranger.ontologyID(range.key) });
    },
    [dispatch, openTab],
  );
};

export const LINKS: Link.Registry = { range: useLink };
