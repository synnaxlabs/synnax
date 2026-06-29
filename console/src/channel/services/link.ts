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
import { useStore } from "react-redux";

import { LinePlot } from "@/layered/service/lineplot";
import { type Link } from "@/layered/service/link";
import { Layout } from "@/layout";
import { Project } from "@/project";
import { Range } from "@/range";
import { type RootState } from "@/store";

export const useLink = (): Link.Handler => {
  const store = useStore<RootState>();
  const placeLayout = Layout.usePlacer();
  return useCallback(
    async ({ client, key }) => {
      const channel = await client.channels.retrieve(key);
      const project = Project.selectOptionalActiveKey(store.getState()) ?? uuid.ZERO;
      const activeRange = Range.selectActiveKey(store.getState()) ?? Range.RECENT_KEY;
      const { key: plotKey, name } = await client.lineplots.create(project, {
        name: `${channel.name} Plot`,
        channels: { y1: [channel.key] },
        ranges: { x1: [activeRange] },
      });
      placeLayout(LinePlot.create({ key: plotKey, name }));
    },
    [store, placeLayout],
  );
};
