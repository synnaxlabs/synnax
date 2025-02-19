// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { MAIN_WINDOW } from "@synnaxlabs/drift";
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import { Icon } from "@synnaxlabs/media";
import { Menu, Mosaic, Text } from "@synnaxlabs/pluto";
import { type direction } from "@synnaxlabs/x";
import { type FC, type ReactElement } from "react";
import { useDispatch, useStore } from "react-redux";

import { useSelectMosaic } from "@/layout/selectors";
import { moveMosaicTab, setFocus, splitMosaicNode } from "@/layout/slice";
import { useOpenInNewWindow } from "@/layout/useOpenInNewWindow";
import { useRemover } from "@/layout/useRemover";

interface MenuItemProps {
  layoutKey: string;
}

const FocusMenuItem = ({ layoutKey }: MenuItemProps) => {
  const dispatch = useDispatch();
  const windowKey = useSelectWindowKey() as string;
  return (
    <Menu.Item
      itemKey="focus"
      startIcon={<Icon.Focus />}
      onClick={() => dispatch(setFocus({ windowKey, key: layoutKey }))}
      trigger={["Control", "L"]}
    >
      Focus
    </Menu.Item>
  );
};

const useMoveIntoMainWindow = () => {
  const store = useStore();
  return (layoutKey: string) => {
    store.dispatch(
      moveMosaicTab({ windowKey: MAIN_WINDOW, tabKey: layoutKey, loc: "center" }),
    );
  };
};

const OpenInNewWindowMenuItem = ({ layoutKey }: MenuItemProps) => {
  const openInNewWindow = useOpenInNewWindow();
  const isMain = useSelectWindowKey() === MAIN_WINDOW;
  if (!isMain) return null;
  return (
    <Menu.Item
      itemKey="openInNewWindow"
      startIcon={<Icon.OpenInNewWindow />}
      onClick={() => openInNewWindow(layoutKey)}
      trigger={["Control", "O"]}
    >
      Open in New Window
    </Menu.Item>
  );
};

const MoveToMainWindowMenuItem = ({ layoutKey }: MenuItemProps) => {
  const moveIntoMainWindow = useMoveIntoMainWindow();
  const windowKey = useSelectWindowKey();
  if (windowKey === MAIN_WINDOW) return null;
  return (
    <Menu.Item
      itemKey="moveIntoMainWindow"
      startIcon={<Icon.OpenInNewWindow />}
      onClick={() => moveIntoMainWindow(layoutKey)}
    >
      Move to Main Window
    </Menu.Item>
  );
};

const CloseMenuItem = ({ layoutKey }: MenuItemProps) => {
  const remove = useRemover();
  return (
    <Menu.Item
      itemKey="close"
      startIcon={<Icon.Close />}
      onClick={() => remove(layoutKey)}
      trigger={["Control", "W"]}
    >
      Close
    </Menu.Item>
  );
};

const RenameMenuItem = ({ layoutKey }: MenuItemProps) => (
  <Menu.Item
    itemKey="rename"
    startIcon={<Icon.Rename />}
    onClick={() => Text.edit(`pluto-tab-${layoutKey}`)}
    trigger={["Control", "E"]}
  >
    Rename
  </Menu.Item>
);

interface SplitMenuItemProps extends MenuItemProps {
  children?: ReactElement;
}

const splitMenuItemFactory = (
  direction: direction.Direction,
): FC<SplitMenuItemProps> => {
  const C = ({ layoutKey, children }: SplitMenuItemProps) => {
    const dispatch = useDispatch();
    const [windowKey, mosaic] = useSelectMosaic();
    if (windowKey == null || mosaic == null) return null;
    const canSplit = Mosaic.canSplit(mosaic, layoutKey);
    if (!canSplit) return null;
    return (
      <>
        {children}
        <Menu.Item
          itemKey={`split${direction}`}
          startIcon={direction === "x" ? <Icon.SplitX /> : <Icon.SplitY />}
          onClick={() =>
            dispatch(splitMosaicNode({ windowKey, tabKey: layoutKey, direction }))
          }
        >
          Split {direction === "x" ? "Horizontally" : "Vertically"}
        </Menu.Item>
      </>
    );
  };
  C.displayName = `Split${direction.toUpperCase()}MenuItem`;
  return C;
};
const SplitXMenuItem = splitMenuItemFactory("x");
const SplitYMenuItem = splitMenuItemFactory("y");

export interface MenuItemsProps {
  layoutKey: string;
}

export const MenuItems = ({ layoutKey }: MenuItemsProps) => (
  <>
    <RenameMenuItem layoutKey={layoutKey} />
    <CloseMenuItem layoutKey={layoutKey} />
    <Menu.Divider />
    <FocusMenuItem layoutKey={layoutKey} />
    <OpenInNewWindowMenuItem layoutKey={layoutKey} />
    <MoveToMainWindowMenuItem layoutKey={layoutKey} />
    <SplitXMenuItem layoutKey={layoutKey}>
      <Menu.Divider />
    </SplitXMenuItem>
    <SplitYMenuItem layoutKey={layoutKey} />
  </>
);
