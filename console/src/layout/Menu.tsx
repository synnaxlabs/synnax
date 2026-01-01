// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { MAIN_WINDOW } from "@synnaxlabs/drift";
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import { Icon, Menu as PMenu, Mosaic, Text } from "@synnaxlabs/pluto";
import { type direction } from "@synnaxlabs/x";
import { type FC, type ReactElement } from "react";
import { useDispatch, useStore } from "react-redux";

import { Menu } from "@/components/menu";
import { useSelectMosaic } from "@/layout/selectors";
import { moveMosaicTab, setFocus, splitMosaicNode } from "@/layout/slice";
import { useOpenInNewWindow } from "@/layout/useOpenInNewWindow";
import { useRemover } from "@/layout/useRemover";
import { Runtime } from "@/runtime";

interface MenuItemProps {
  layoutKey: string;
}

const FocusMenuItem = ({ layoutKey }: MenuItemProps): ReactElement => {
  const dispatch = useDispatch();
  const windowKey = useSelectWindowKey() as string;
  return (
    <PMenu.Item
      itemKey="focus"
      onClick={() => dispatch(setFocus({ windowKey, key: layoutKey }))}
      trigger={["Control", "L"]}
    >
      <Icon.Focus />
      Focus
    </PMenu.Item>
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

const OpenInNewWindowMenuItem = ({ layoutKey }: MenuItemProps): ReactElement | null => {
  const openInNewWindow = useOpenInNewWindow();
  const isMain = useSelectWindowKey() === MAIN_WINDOW;
  if (!isMain) return null;
  return (
    <PMenu.Item
      itemKey="openInNewWindow"
      onClick={() => openInNewWindow(layoutKey)}
      trigger={["Control", "O"]}
      triggerIndicator
    >
      <Icon.OpenInNewWindow />
      Open in new window
    </PMenu.Item>
  );
};

const MoveToMainWindowMenuItem = ({
  layoutKey,
}: MenuItemProps): ReactElement | null => {
  const moveIntoMainWindow = useMoveIntoMainWindow();
  const windowKey = useSelectWindowKey();
  if (windowKey === MAIN_WINDOW) return null;
  return (
    <PMenu.Item
      itemKey="moveIntoMainWindow"
      onClick={() => moveIntoMainWindow(layoutKey)}
    >
      <Icon.OpenInNewWindow />
      Move to main window
    </PMenu.Item>
  );
};

const CloseMenuItem = ({ layoutKey }: MenuItemProps): ReactElement => {
  const remove = useRemover();
  return (
    <PMenu.Item
      itemKey="close"
      onClick={() => remove(layoutKey)}
      trigger={["Control", "W"]}
      triggerIndicator
    >
      <Icon.Close />
      Close
    </PMenu.Item>
  );
};

const RenameMenuItem = ({ layoutKey }: MenuItemProps): ReactElement => (
  <PMenu.Item
    itemKey="rename"
    onClick={() => Text.edit(`pluto-tab-${layoutKey}`)}
    trigger={["Control", "E"]}
    triggerIndicator
  >
    <Icon.Rename />
    Rename
  </PMenu.Item>
);

interface SplitMenuItemProps extends MenuItemProps {}

const splitMenuItemFactory = (
  direction: direction.Direction,
): FC<SplitMenuItemProps> => {
  const C = ({ layoutKey }: SplitMenuItemProps) => {
    const dispatch = useDispatch();
    const [windowKey, mosaic] = useSelectMosaic();
    if (windowKey == null || mosaic == null) return null;
    const canSplit = Mosaic.canSplit(mosaic, layoutKey);
    if (!canSplit) return null;
    return (
      <PMenu.Item
        itemKey={`split${direction}`}
        onClick={() =>
          dispatch(splitMosaicNode({ windowKey, tabKey: layoutKey, direction }))
        }
      >
        {direction === "x" ? <Icon.SplitX /> : <Icon.SplitY />}
        Split {direction === "x" ? "horizontally" : "vertically"}
      </PMenu.Item>
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

export const MenuItems = ({ layoutKey }: MenuItemsProps): ReactElement => (
  <>
    <RenameMenuItem layoutKey={layoutKey} />
    <CloseMenuItem layoutKey={layoutKey} />
    <PMenu.Divider />
    <FocusMenuItem layoutKey={layoutKey} />
    {Runtime.ENGINE === "tauri" && <OpenInNewWindowMenuItem layoutKey={layoutKey} />}
    <MoveToMainWindowMenuItem layoutKey={layoutKey} />
    <PMenu.Divider />
    <SplitXMenuItem layoutKey={layoutKey} />
    <SplitYMenuItem layoutKey={layoutKey} />
    <PMenu.Divider />
    <Menu.ReloadConsoleItem />
  </>
);
