// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useEffect } from "react";

import { Align } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { Layout } from "@/layout";
import {
  NavBottom,
  NavDrawer,
  NavLeft,
  NavRight,
  NavTop,
} from "@/layouts/LayoutMain/Nav";

import "@/layouts/LayoutMain/LayoutMain.css";

/**
 * The center of it all. This is the main layout for the Delta UI. Try to keep this
 * component as simple, presentational, and navigatable as possible.
 */
export const LayoutMain = (): ReactElement => {
  const d = useDispatch();
  useEffect(() => {
    d(Layout.maybeCreateGetStartedTab());
  }, []);
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
              <Layout.Mosaic />
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
