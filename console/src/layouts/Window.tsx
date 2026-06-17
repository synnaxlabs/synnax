// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layouts/Window.css";

import { MAIN_WINDOW, setWindowProps } from "@synnaxlabs/drift";
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import { Flex, Haul, Menu, OS } from "@synnaxlabs/pluto";
import { getCurrentWindow } from "@tauri-apps/api/window";
import { memo, type ReactElement, useEffect } from "react";
import { useDispatch } from "react-redux";

import { ContextMenu } from "@/components";
import { CSS } from "@/css";
import { Aux } from "@/layouts/Aux";
import { Main } from "@/layouts/Main";
import { Modals } from "@/modals";
import { Runtime } from "@/runtime";

export const DefaultContextMenu = (): ReactElement => (
  <ContextMenu.Menu>
    <ContextMenu.ReloadConsoleItem />
  </ContextMenu.Menu>
);

const menu = () => <DefaultContextMenu />;

const WindowInternal = (): ReactElement | null => {
  const currLabel = Runtime.ENGINE === "tauri" ? getCurrentWindow().label : MAIN_WINDOW;
  const isMain = currLabel === MAIN_WINDOW;
  const resolvedKey = useSelectWindowKey(currLabel);
  const windowKey = isMain ? MAIN_WINDOW : resolvedKey;
  const os = OS.use({ default: "Windows" });
  const dispatch = useDispatch();

  useEffect(() => {
    if (windowKey == null) return;
    dispatch(
      setWindowProps({
        key: windowKey,
        visible: true,
        minimized: false,
        decorations: os !== "Windows",
      }),
    );
  }, [os, windowKey]);

  const menuProps = Menu.useContextMenu();
  const ctx = Haul.useContext();
  const dragging = Haul.useDraggingRef();
  const handleDragOver = (event: React.DragEvent<HTMLDivElement>) => {
    if (Haul.isFileDrag(event, dragging.current))
      ctx?.start(Haul.ZERO_ITEM, [Haul.FILE]);
  };

  if (windowKey == null) return null;

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
      <Modals.Provider>
        <Menu.ContextMenu menu={menu} {...menuProps}>
          {isMain ? <Main /> : <Aux />}
        </Menu.ContextMenu>
      </Modals.Provider>
    </Flex.Box>
  );
};

export const Window = memo(WindowInternal);
Window.displayName = "Window";
