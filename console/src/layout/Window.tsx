// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layout/Window.css";

import { MAIN_WINDOW, setWindowProps } from "@synnaxlabs/drift";
import { useSelectWindowAttribute, useSelectWindowKey } from "@synnaxlabs/drift/react";
import { Logo } from "@synnaxlabs/media";
import { Align, Haul, Menu as PMenu, Nav, OS, Text } from "@synnaxlabs/pluto";
import { getCurrentWindow } from "@tauri-apps/api/window";
import { memo, useEffect } from "react";
import { useDispatch } from "react-redux";

import { Menu } from "@/components";
import { CSS } from "@/css";
import { Content } from "@/layout/Content";
import { Controls } from "@/layout/Controls";
import { useSelect } from "@/layout/selectors";
import { type WindowProps } from "@/layout/slice";

interface NavTopProps extends Pick<WindowProps, "showTitle" | "navTop"> {
  title: string;
}

const NavTop = ({ title, showTitle = true, navTop }: NavTopProps) => {
  const os = OS.use();
  const isWindowsOS = os === "Windows";
  return !navTop ? null : (
    <Nav.Bar
      className="console-main-nav-top"
      location="top"
      size="6rem"
      data-tauri-drag-region
    >
      <Nav.Bar.Start className="console-main-nav-top__start" data-tauri-drag-region>
        <Controls
          className="console-controls--macos"
          visibleIfOS="macOS"
          forceOS={os}
        />
        {isWindowsOS && <Logo className="console-main-nav-top__logo" />}
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
      {isWindowsOS && (
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

export const DefaultContextMenu = () => (
  <PMenu.Menu>
    <Menu.HardReloadItem />
  </PMenu.Menu>
);

const WindowInternal = () => {
  const currLabel = getCurrentWindow().label;
  const isMain = currLabel === MAIN_WINDOW;
  let win = useSelectWindowKey(currLabel) ?? "";
  if (isMain) win = MAIN_WINDOW;
  const layout = useSelect(win);
  const os = OS.use({ default: "Windows" });
  const dispatch = useDispatch();
  useEffect(() => {
    if (layout?.key == null) return;
    dispatch(
      setWindowProps({
        key: layout?.key,
        visible: true,
        minimized: false,
        decorations: os !== "Windows",
      }),
    );
  }, [os, layout?.key]);

  const menuProps = PMenu.useContextMenu();
  const maximized = useSelectWindowAttribute(win, "maximized") ?? false;
  const ctx = Haul.useContext();
  const dragging = Haul.useDraggingRef();

  const handleDragOver = (event: React.DragEvent<HTMLDivElement>) => {
    if (Haul.isFileDrag(event, dragging.current))
      ctx?.start(Haul.ZERO_ITEM, [Haul.FILE]);
  };

  return layout == null ? null : (
    <PMenu.ContextMenu menu={() => <DefaultContextMenu />} {...menuProps}>
      <Align.Space
        empty
        className={CSS(
          CSS.B("main"),
          CSS.BM("main", os.toLowerCase()),
          maximized && CSS.BM("main", "maximized"),
        )}
        onDragOver={handleDragOver}
      >
        <NavTop title={layout.name} {...layout.window} />
        <Content layoutKey={layout.key} />
      </Align.Space>
    </PMenu.ContextMenu>
  );
};

export const Window = memo(WindowInternal);
Window.displayName = "Window";
