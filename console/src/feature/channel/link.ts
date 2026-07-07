// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { uuid } from "@synnaxlabs/x";
import { useCallback } from "react";

import { LinePlot } from "@/platform/lineplot";
import { type Link } from "@/platform/link";
import { Session } from "@/session";

export const useLink = (): Link.Handler => {
  const create = LinePlot.useCreate();
  const getOptionalSelected = Session.Project.useGetOptionalSelected();
  const getSelectedKey = Session.Range.useGetSelectedKey();
  return useCallback(
    async ({ client, key }) => {
      const channel = await client.channels.retrieve(key);
      const project = getOptionalSelected() ?? uuid.ZERO;
      const activeRange = getSelectedKey() ?? Session.Range.RECENT_KEY;
      create({
        name: `${channel.name} Plot`,
        channels: { y1: [channel.key] },
        ranges: { x1: [activeRange] },
        project,
      });
    },
    [create, getOptionalSelected, getSelectedKey],
  );
};

export const LINKS: Link.Registry = { channel: useLink };
