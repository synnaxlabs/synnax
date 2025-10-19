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
import { ContextMenu as PContextMenu, Icon, Mosaic, Text } from "@synnaxlabs/pluto";
import { type direction } from "@synnaxlabs/x";
import { type FC, type ReactElement } from "react";
import { useDispatch, useStore } from "react-redux";

import { ContextMenu } from "@/components/context-menu";
import { useSelectMosaic } from "@/layout/selectors";
import { moveMosaicTab, setFocus, splitMosaicNode } from "@/layout/slice";
import { useOpenInNewWindow } from "@/layout/useOpenInNewWindow";
import { useRemover } from "@/layout/useRemover";
import { Runtime } from "@/runtime";

interface MenuItemProps extends Pick<PContextMenu.ItemProps, "showBottomDivider"> {
  layoutKey: string;
}

const FocusMenuItem = ({
  layoutKey,
  showBottomDivider,
}: MenuItemProps): ReactElement => {
  const dispatch = useDispatch();
  const windowKey = useSelectWindowKey() as string;
  return (
    <PContextMenu.Item
      onClick={() => dispatch(setFocus({ windowKey, key: layoutKey }))}
      trigger={["Control", "L"]}
      showBottomDivider={showBottomDivider}
    >
      <Icon.Focus />
      Focus
    </PContextMenu.Item>
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

const OpenInNewWindowMenuItem = ({
  layoutKey,
  showBottomDivider,
}: MenuItemProps): ReactElement | null => {
  const openInNewWindow = useOpenInNewWindow();
  const isMain = useSelectWindowKey() === MAIN_WINDOW;
  if (!isMain) return null;
  return (
    <PContextMenu.Item
      onClick={() => openInNewWindow(layoutKey)}
      trigger={["Control", "O"]}
      triggerIndicator
      showBottomDivider={showBottomDivider}
    >
      <Icon.OpenInNewWindow />
      Open in new window
    </PContextMenu.Item>
  );
};

const MoveToMainWindowMenuItem = ({
  layoutKey,
  showBottomDivider,
}: MenuItemProps): ReactElement | null => {
  const moveIntoMainWindow = useMoveIntoMainWindow();
  const windowKey = useSelectWindowKey();
  if (windowKey === MAIN_WINDOW) return null;
  return (
    <PContextMenu.Item
      onClick={() => moveIntoMainWindow(layoutKey)}
      showBottomDivider={showBottomDivider}
    >
      <Icon.OpenInNewWindow />
      Move to main window
    </PContextMenu.Item>
  );
};

const CloseMenuItem = ({
  layoutKey,
  showBottomDivider,
}: MenuItemProps): ReactElement => {
  const remove = useRemover();
  return (
    <PContextMenu.Item
      onClick={() => remove(layoutKey)}
      trigger={["Control", "W"]}
      triggerIndicator
      showBottomDivider={showBottomDivider}
    >
      <Icon.Close />
      Close
    </PContextMenu.Item>
  );
};

const RenameMenuItem = ({
  layoutKey,
  showBottomDivider,
}: MenuItemProps): ReactElement => (
  <PContextMenu.Item
    onClick={() => Text.edit(`pluto-tab-${layoutKey}`)}
    trigger={["Control", "E"]}
    triggerIndicator
    showBottomDivider={showBottomDivider}
  >
    <Icon.Rename />
    Rename
  </PContextMenu.Item>
);

interface SplitMenuItemProps extends MenuItemProps {}

const splitMenuItemFactory = (
  direction: direction.Direction,
): FC<SplitMenuItemProps> => {
  const C = ({ layoutKey, showBottomDivider }: SplitMenuItemProps) => {
    const dispatch = useDispatch();
    const [windowKey, mosaic] = useSelectMosaic();
    if (windowKey == null || mosaic == null) return null;
    const canSplit = Mosaic.canSplit(mosaic, layoutKey);
    if (!canSplit) return null;
    return (
      <PContextMenu.Item
        onClick={() =>
          dispatch(splitMosaicNode({ windowKey, tabKey: layoutKey, direction }))
        }
        showBottomDivider={showBottomDivider}
      >
        {direction === "x" ? <Icon.SplitX /> : <Icon.SplitY />}
        Split {direction === "x" ? "horizontally" : "vertically"}
      </PContextMenu.Item>
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
    <CloseMenuItem layoutKey={layoutKey} showBottomDivider />
    <FocusMenuItem layoutKey={layoutKey} />
    {Runtime.ENGINE === "tauri" && <OpenInNewWindowMenuItem layoutKey={layoutKey} />}
    <MoveToMainWindowMenuItem layoutKey={layoutKey} showBottomDivider />
    <SplitXMenuItem layoutKey={layoutKey} />
    <SplitYMenuItem layoutKey={layoutKey} showBottomDivider />
    <ContextMenu.ReloadConsoleItem />
  </>
);
