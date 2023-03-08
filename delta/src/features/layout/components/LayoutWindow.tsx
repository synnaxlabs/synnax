// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useSelectWindowKey } from "@synnaxlabs/drift";
import { Nav, Space } from "@synnaxlabs/pluto";
import { appWindow } from "@tauri-apps/api/window";

import { useSelectLayout } from "../store";

import { LayoutContent } from "./LayoutContent";

import { Controls } from "@/components";

import "./LayoutWindow.css";

export const NavTop = (): JSX.Element => (
  <Nav.Bar data-tauri-drag-region location="top" size={"6rem"}>
    <Nav.Bar.Start className="delta-main-nav-top__start">
      <Controls className="delta-macos-controls" visibleIfOS="MacOS" />
    </Nav.Bar.Start>
    <Nav.Bar.End style={{ padding: "0 2rem" }}>
      <Controls className="delta-windows-controls" visibleIfOS="Windows" />
    </Nav.Bar.End>
  </Nav.Bar>
);

export const LayoutWindow = (): JSX.Element => {
  const { label } = appWindow;
  const key = useSelectWindowKey(label);
  console.log(key);

  const layout = useSelectLayout(key);
  if (key == null) return <h1>{label}</h1>;
  const content = <LayoutContent layoutKey={key} />;
  if (layout?.window?.navTop === true)
    return (
      <Space empty className="delta-main">
        <NavTop />
        {content}
      </Space>
    );
  return content;
};
