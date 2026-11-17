// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/app/window/Window.css";

import { Drift } from "@synnaxlabs/drift";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Haul } from "@synnaxlabs/lyra/haul";
import { Menu } from "@synnaxlabs/lyra/menu";
import { OS } from "@synnaxlabs/pluto";
import { memo, type ReactElement, useEffect } from "react";

import { Notifications } from "@/app/notifications";
import { Primary } from "@/app/window/Primary";
import { Secondary } from "@/app/window/Secondary";
import { Embedded } from "@/feature/embedded";
import { ContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Version } from "@/platform/version";
import { Session } from "@/session";

const DefaultContextMenu = (): ReactElement => (
  <ContextMenu.Menu>
    <ContextMenu.ReloadConsoleItem />
  </ContextMenu.Menu>
);

const menu = Component.renderProp(DefaultContextMenu);

const INSTALL_MIDDLEWARE = DESKTOP ? Embedded.installMiddleware : undefined;

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

  const stack = (
    <Modals.Stack>
      <Notifications.Feed />
      <Menu.ContextMenu menu={menu} {...menuProps}>
        {isMain ? <Primary /> : <Secondary />}
      </Menu.ContextMenu>
    </Modals.Stack>
  );

  return (
    <Flex.Box
      full
      empty
      className={CSS.cls(
        CSS.B("main"),
        CSS.M(`os-${os.toLowerCase()}`),
        isMain && CSS.BM("main", "primary"),
        menuProps.className,
      )}
      onDragOver={handleDragOver}
      onContextMenu={menuProps.open}
    >
      {isMain ? (
        <Version.Provider installMiddleware={INSTALL_MIDDLEWARE}>
          {stack}
        </Version.Provider>
      ) : (
        stack
      )}
    </Flex.Box>
  );
});
Window.displayName = "Window";
