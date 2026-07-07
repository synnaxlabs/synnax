// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/app/Window.css";

import { MAIN_WINDOW, setWindowProps } from "@synnaxlabs/drift";
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import { Component, Flex, Haul, Menu, OS } from "@synnaxlabs/pluto";
import { getCurrentWindow } from "@tauri-apps/api/window";
import { memo, type ReactElement, useEffect } from "react";

import { Aux } from "@/app/Aux";
import { Main } from "@/app/main/Main";
import { ContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Session } from "@/session";

export const DefaultContextMenu = (): ReactElement => (
  <ContextMenu.Menu>
    <ContextMenu.ReloadConsoleItem />
  </ContextMenu.Menu>
);

const menu = Component.renderProp(DefaultContextMenu);

// WindowInternal is the shell every OS window renders: the main window hosts
// Main (nav + panel workspace); every other window hosts Aux (a bare panel
// viewport). Pre-render windows stay hidden until they are handed a panel.
const WindowInternal = (): ReactElement | null => {
  const currLabel =
    Session.Runtime.ENGINE === "tauri" ? getCurrentWindow().label : MAIN_WINDOW;
  const isMain = currLabel === MAIN_WINDOW;
  const windowKey = useSelectWindowKey(currLabel);
  const selectedPanel = Session.Panel.useSelectSelected();
  const hasContent = isMain || selectedPanel != null;
  const os = OS.use({ default: "Windows" });
  const dispatch = Session.useDispatch();
  useEffect(() => {
    if (windowKey == null || !hasContent) return;
    dispatch(
      setWindowProps({
        key: windowKey,
        visible: true,
        minimized: false,
        decorations: os !== "Windows",
      }),
    );
  }, [os, windowKey, hasContent]);

  const menuProps = Menu.useContextMenu();
  const ctx = Haul.useContext();
  const dragging = Haul.useDraggingRef();

  const handleDragOver = (event: React.DragEvent<HTMLDivElement>) => {
    if (Haul.isFileDrag(event, dragging.current))
      ctx?.start(Haul.ZERO_ITEM, [Haul.FILE]);
  };

  return (
    <Flex.Box
      empty
      className={CSS(
        CSS.B("main"),
        CSS.M(`os-${os.toLowerCase()}`),
        menuProps.className,
      )}
      onDragOver={handleDragOver}
      onContextMenu={menuProps.open}
    >
      <Modals.Stack />
      <Menu.ContextMenu menu={menu} {...menuProps}>
        {isMain ? <Main /> : <Aux />}
      </Menu.ContextMenu>
    </Flex.Box>
  );
};

export const Window = memo(WindowInternal);
Window.displayName = "Window";
