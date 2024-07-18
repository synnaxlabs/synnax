// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layout/Window.css";

import {
  setWindowDecorations,
  setWindowMinimized,
  setWindowVisible,
} from "@synnaxlabs/drift";
import { useSelectWindowAttribute, useSelectWindowKey } from "@synnaxlabs/drift/react";
import { Logo } from "@synnaxlabs/media";
import { Align, Haul, Menu as PMenu, Nav, OS, Text } from "@synnaxlabs/pluto";
import { runtime } from "@synnaxlabs/x";
import { getCurrent } from "@tauri-apps/api/window";
import { type ReactElement, useEffect } from "react";
import { useDispatch } from "react-redux";

import { Controls } from "@/components";
import { Menu } from "@/components/menu";
import { CSS } from "@/css";
import { Content } from "@/layout/Content";
import { useSelect } from "@/layout/selectors";
import { WindowProps } from "@/layout/slice";

export interface NavTopProps extends Pick<WindowProps, "showTitle" | "navTop"> {
  title: string;
}

export const NavTop = ({
  title,
  showTitle = true,
  navTop,
}: NavTopProps): ReactElement | null => {
  const os = OS.use();
  if (!navTop) return null;

  return (
    <Nav.Bar
      className="console-main-nav-top"
      location="top"
      size={"6rem"}
      data-tauri-drag-region
    >
      <Nav.Bar.Start className="console-main-nav-top__start" data-tauri-drag-region>
        <Controls
          className="console-controls--macos"
          visibleIfOS="MacOS"
          forceOS={os}
        />
        {os === "Windows" && <Logo className="console-main-nav-top__logo" />}
      </Nav.Bar.Start>
      {showTitle && (
        <Nav.Bar.AbsoluteCenter data-tauri-drag-region>
          <Text.Text
            className="console-main-nav-top__title"
            level="p"
            shade={7}
            weight={450}
            data-tauri-drag-region
          >
            {title}
          </Text.Text>
        </Nav.Bar.AbsoluteCenter>
      )}
      {os === "Windows" && (
        <Nav.Bar.End data-tauri-drag-region>
          <Controls
            className="console-controls--windows"
            visibleIfOS="Windows"
            forceOS={os}
          />
        </Nav.Bar.End>
      )}
    </Nav.Bar>
  );
};

export const DefaultContextMenu = (): ReactElement => (
  <PMenu.Menu>
    <Menu.HardReloadItem />
  </PMenu.Menu>
);

export const Window = (): ReactElement | null => {
  const win = useSelectWindowKey(getCurrent().label) ?? "";
  const layout = useSelect(win);
  const os = OS.use({ default: "Windows" }) as runtime.OS;
  const dispatch = useDispatch();

  useEffect(() => {
    if (layout?.key != null) {
      dispatch(setWindowVisible({ key: layout.key, value: true }));
      dispatch(setWindowMinimized({ key: layout.key, value: false }));
    }
    if (os === "Windows") dispatch(setWindowDecorations({ value: false }));
  }, [os, layout?.key]);

  const menuProps = PMenu.useContextMenu();
  const maximized = useSelectWindowAttribute(win, "maximized") ?? false;
  const ctx = Haul.useContext();
  const dragging = Haul.useDraggingRef();
  if (layout == null) return null;
  const content = <Content layoutKey={layout.key} />;

  return (
    <PMenu.ContextMenu menu={() => <DefaultContextMenu />} {...menuProps}>
      <Align.Space
        empty
        className={CSS(
          CSS.B("main"),
          CSS.BM("main", os.toLowerCase()),
          maximized && CSS.BM("main", "maximized"),
        )}
        onDragOver={(event) => {
          if (Haul.isFileDrag(event, dragging.current))
            ctx?.start(Haul.ZERO_ITEM, [Haul.FILE]);
        }}
      >
        <NavTop title={layout.name} {...layout.window} />
        {content}
      </Align.Space>
    </PMenu.ContextMenu>
  );
};
