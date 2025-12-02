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
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import { ContextMenu as PContextMenu, Flex, Haul, OS } from "@synnaxlabs/pluto";
import { getCurrentWindow } from "@tauri-apps/api/window";
import { memo, type ReactElement, useEffect } from "react";
import { useDispatch } from "react-redux";

import { ContextMenu } from "@/components";
import { CSS } from "@/css";
import { Content } from "@/layout/Content";
import { Modals } from "@/layout/Modals";
import { useSelect } from "@/layout/selectors";
import { Runtime } from "@/runtime";

export const DefaultContextMenu = (): ReactElement => <ContextMenu.ReloadConsoleItem />;

const WindowInternal = (): ReactElement | null => {
  const currLabel = Runtime.ENGINE === "tauri" ? getCurrentWindow().label : MAIN_WINDOW;
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

  const contextMenuProps = PContextMenu.use();
  const ctx = Haul.useContext();
  const dragging = Haul.useDraggingRef();

  const handleDragOver = (event: React.DragEvent<HTMLDivElement>) => {
    if (Haul.isFileDrag(event, dragging.current))
      ctx?.start(Haul.ZERO_ITEM, [Haul.FILE]);
  };
  if (layout == null) return null;

  return (
    <Flex.Box
      empty
      className={CSS(
        CSS.B("main"),
        CSS.M(`os-${os.toLowerCase()}`),
        contextMenuProps.className,
      )}
      onDragOver={handleDragOver}
      onContextMenu={contextMenuProps.open}
    >
      <Modals />
      <PContextMenu.ContextMenu
        menu={() => <DefaultContextMenu />}
        {...contextMenuProps}
      >
        <Content layoutKey={layout.key} />
      </PContextMenu.ContextMenu>
    </Flex.Box>
  );
};

export const Window = memo(WindowInternal);
Window.displayName = "Window";
