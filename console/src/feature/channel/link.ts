// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback } from "react";

import { LinePlot } from "@/platform/lineplot";
import { type Link } from "@/platform/link";
import { Session } from "@/session";

export const useLink = (): Link.Handler => {
  const create = LinePlot.useCreate();
  const getSelectedKey = Session.Range.useGetSelectedKey();
  return useCallback(
    async ({ client, key }) => {
      const channel = await client.channels.retrieve(key);
      const activeRange = getSelectedKey() ?? Session.Range.RECENT_KEY;
      create({
        name: `${channel.name} Plot`,
        channels: { y1: [channel.key] },
        ranges: { x1: [activeRange] },
      });
    },
    [create, getSelectedKey],
  );
};

export const LINKS: Link.Registry = { channel: useLink };
