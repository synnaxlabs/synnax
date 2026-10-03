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

export const useLink = (): Link.Handler => {
  const create = LinePlot.useCreate();
  return useCallback(
    async ({ client, key }) => {
      const channel = await client.channels.retrieve(key);
      create({ name: `${channel.name} Plot`, channels: { y1: [channel.key] } });
    },
    [create],
  );
};

export const LINKS: Link.Registry = { channel: useLink };
