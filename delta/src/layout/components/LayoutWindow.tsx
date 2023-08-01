// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useEffect } from "react";

import {
  setWindowDecorations,
  useSelectWindowAttribute,
  useSelectWindowKey,
} from "@synnaxlabs/drift";
import { Logo } from "@synnaxlabs/media";
import { Nav, Space, useOS, CSS as PCSS, Menu as PMenu } from "@synnaxlabs/pluto";
import { appWindow } from "@tauri-apps/api/window";
import { useDispatch } from "react-redux";

import { LayoutContent } from "./LayoutContent";

import { Controls, Menu } from "@/components";
import { CSS } from "@/css";
import { useSelectLayout } from "@/layout/store";

import "@/layout/components/LayoutWindow.css";

export const NavTop = (): ReactElement => {
  const os = useOS();
  return (
    <Nav.Bar data-tauri-drag-region location="top" size={"6rem"}>
      <Nav.Bar.Start className="delta-main-nav-top__start">
        <Controls className="delta-controls--macos" visibleIfOS="MacOS" />
        {os === "Windows" && <Logo className="delta-main-nav-top__logo" />}
      </Nav.Bar.Start>
      <Nav.Bar.End>
        <Controls className="delta-controls--windows" visibleIfOS="Windows" />
      </Nav.Bar.End>
    </Nav.Bar>
  );
};

export const DefaultContextMenu = (): ReactElement => (
  <PMenu>
    <Menu.Item.HardReload />
  </PMenu>
);

export const LayoutWindow = (): ReactElement | null => {
  const { label } = appWindow;
  const win = useSelectWindowKey(label);
  const layout = useSelectLayout(win ?? "");
  const os = useOS();
  const dispatch = useDispatch();
  useEffect(() => {
    if (os === "Windows") {
      applyWindowsBorders();
      dispatch(setWindowDecorations({ value: false }));
    }
  }, [os]);
  const menuProps = PMenu.useContextMenu();
  const maximized = useSelectWindowAttribute(label, "maximized") ?? false;
  if (layout == null) return null;
  const content = <LayoutContent layoutKey={layout.key} />;
  return (
    <PMenu.ContextMenu menu={() => <DefaultContextMenu />} {...menuProps}>
      <Space
        empty
        className={CSS(
          CSS.B("main"),
          CSS.BM("main", os?.toLowerCase() as string),
          maximized && CSS.BM("main", "maximized")
        )}
      >
        {layout?.window?.navTop === true && <NavTop />}
        {content}
        <div className="delta-background" />
        <div className="delta-border" />
      </Space>
    </PMenu.ContextMenu>
  );
};


