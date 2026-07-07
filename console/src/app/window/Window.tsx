// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/app/Window.css";

import { Drift } from "@synnaxlabs/drift";
import { Component, Flex, Haul, Menu, OS } from "@synnaxlabs/pluto";
import { memo, type ReactElement, useEffect } from "react";

import { Aux } from "@/app/window/Aux";
import { Main } from "@/app/window/Main";
import { ContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Session } from "@/session";

const DefaultContextMenu = (): ReactElement => (
  <ContextMenu.Menu>
    <ContextMenu.ReloadConsoleItem />
  </ContextMenu.Menu>
);

const menu = Component.renderProp(DefaultContextMenu);

export const Window = memo((): ReactElement | null => {
  const isMain = Session.Runtime.isMainWindow();
  const os = OS.use({ default: "Windows" });
  const dispatch = Session.useDispatch();
  useEffect(() => {
    dispatch(
      Drift.setWindowProps({
        visible: true,
        minimized: false,
        decorations: os !== "Windows",
      }),
    );
  }, [os]);

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
});
Window.displayName = "Window";
