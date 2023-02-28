// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Nav, Space } from "@synnaxlabs/pluto";
import { appWindow } from "@tauri-apps/api/window";

import { useSelectLayout } from "../store";

import { LayoutContent } from "./LayoutContent";

import "./LayoutWindow.css";

export const NavTop = (): JSX.Element => (
  <Nav.Bar data-tauri-drag-region location="top" />
);

export const LayoutWindow = (): JSX.Element => {
  const { label: key } = appWindow;
  const layout = useSelectLayout(key);
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
