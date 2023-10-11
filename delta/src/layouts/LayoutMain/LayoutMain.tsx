// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useEffect, useCallback } from "react";

import { ontology } from "@synnaxlabs/client";
import { Align, Synnax } from "@synnaxlabs/pluto";
import { type location } from "@synnaxlabs/x";
import { useDispatch, useStore } from "react-redux";

import { Layout } from "@/layout";
import { usePlacer } from "@/layout/hooks";
import {
  NavBottom,
  NavDrawer,
  NavLeft,
  NavRight,
  NavTop,
} from "@/layouts/LayoutMain/Nav";
import { SERVICES } from "@/services";
import { type RootStore } from "@/store";
import { Vis } from "@/vis";

import "@/layouts/LayoutMain/LayoutMain.css";

const createNewVis = (
  placer: Layout.Placer,
  mosaicKey: number,
  loc: location.Location
): void => {
  placer(Vis.create({ tab: { mosaicKey, location: loc }, location: "mosaic" }));
};

/**
 * The center of it all. This is the main layout for the Delta UI. Try to keep this
 * component as simple, presentational, and navigatable as possible.
 */
export const LayoutMain = (): ReactElement => {
  const d = useDispatch();
  useEffect(() => {
    d(Layout.maybeCreateGetStartedTab());
  }, []);

  const client = Synnax.use();
  const store = useStore();
  const placer = usePlacer();

  const handleCreate = useCallback(
    (mosaicKey: number, location: location.Location, tabKeys?: string[]) => {
      if (tabKeys == null) return createNewVis(placer, mosaicKey, location);
      tabKeys.forEach((tabKey) => {
        const res = ontology.stringIDZ.safeParse(tabKey);
        if (res.success) {
          const id = res.data;
          if (client == null) return;
          SERVICES[id.type].onMosaicDrop?.({
            client,
            store: store as RootStore,
            id,
            nodeKey: mosaicKey,
            location,
            placeLayout: placer,
          });
        } else placer(Vis.create({ tab: { mosaicKey, location }, location: "mosaic" }));
      });
    },
    [placer, store, client]
  );

  return (
    <>
      <NavTop />
      <Align.Space className="delta-main-fixed--y" direction="x" empty>
        <NavLeft />
        <Align.Space
          className="delta-main-content-drawers delta-main-fixed--y delta-main-fixed--x"
          empty
        >
          <Align.Space className="delta-main--driven" direction="x" empty>
            <NavDrawer location="left" />
            <main className="delta-main--driven" style={{ position: "relative" }}>
              <Layout.Mosaic onCreate={handleCreate} />
            </main>
            <NavDrawer location="right" />
          </Align.Space>
          <NavDrawer location="bottom" />
        </Align.Space>
        <NavRight />
      </Align.Space>
      <NavBottom />
    </>
  );
};
