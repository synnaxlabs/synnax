// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useEffect } from "react";

import { Align } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { NavDrawer } from "@/components/nav/Nav";
import { Layout } from "@/layout";
import { NavBottom, NavLeft, NavRight, NavTop } from "@/layouts/LayoutMain/Nav";
import { Mosaic } from "@/layouts/mosaic";

import "@/layouts/LayoutMain/LayoutMain.css";

/**
 * The center of it all. This is the main layout for the Synnax Console. Try to keep this
 * component as simple, presentational, and navigatable as possible.
 */
export const LayoutMain = (): ReactElement => {
  const d = useDispatch();
  useEffect(() => {
    d(Layout.maybeCreateGetStartedTab());
  }, []);

  // Cluster.useLocalServer();

  return (
    <>
      <NavTop />
      <Align.Space className="console-main-fixed--y" direction="x" empty>
        <NavLeft />
        <Align.Space
          className="console-main-content-drawers console-main-fixed--y console-main-fixed--x"
          empty
        >
          <Align.Space className="console-main--driven" direction="x" empty>
            <NavDrawer location="left" />
            <main className="console-main--driven" style={{ position: "relative" }}>
              <Mosaic.Mosaic />
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
