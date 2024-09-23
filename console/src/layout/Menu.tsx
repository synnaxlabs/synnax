import { MAIN_WINDOW } from "@synnaxlabs/drift";
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import { Icon } from "@synnaxlabs/media";
import { Menu, Mosaic, Text } from "@synnaxlabs/pluto";
import { direction } from "@synnaxlabs/x";
import { FC, ReactElement } from "react";
import { useDispatch, useStore } from "react-redux";

import { usePlacer, useRemover } from "@/layout/hooks";
import { useSelectMosaic } from "@/layout/selectors";
import {
  createMosaicWindow,
  moveMosaicTab,
  setFocus,
  splitMosaicNode,
} from "@/layout/slice";

export interface FocusMenuItemProps {
  layoutKey: string;
}

export const FocusMenuItem = ({ layoutKey }: FocusMenuItemProps): ReactElement => {
  const d = useDispatch();
  const windowKey = useSelectWindowKey() as string;
  return (
    <Menu.Item
      itemKey="focus"
      startIcon={<Icon.Focus />}
      onClick={() => d(setFocus({ windowKey: windowKey, key: layoutKey }))}
      trigger={["Control", "F"]}
    >
      Focus
    </Menu.Item>
  );
};

export const useOpenInNewWindow = () => {
  const d = useDispatch();
  const placer = usePlacer();
  return (layoutKey: string) => {
    const { key } = placer(createMosaicWindow({}));
    d(
      moveMosaicTab({
        windowKey: key,
        key: 1,
        tabKey: layoutKey,
        loc: "center",
      }),
    );
  };
};

export const useMoveIntoMainWindow = () => {
  const s = useStore();
  return (layoutKey: string) => {
    s.dispatch(
      moveMosaicTab({
        windowKey: MAIN_WINDOW,
        tabKey: layoutKey,
        loc: "center",
      }),
    );
  };
};

export const OpenInNewWindowMenuItem = ({
  layoutKey,
}: FocusMenuItemProps): ReactElement | null => {
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

export const MoveToMainWindowMenuItem = ({
  layoutKey,
}: FocusMenuItemProps): ReactElement | null => {
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

export const CloseMenuItem = ({ layoutKey }: FocusMenuItemProps): ReactElement => {
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

export const RenameMenuItem = ({ layoutKey }: FocusMenuItemProps): ReactElement => {
  return (
    <Menu.Item
      itemKey="rename"
      startIcon={<Icon.Rename />}
      onClick={() => Text.edit(`pluto-tab-${layoutKey}`)}
      trigger={["Control", "R"]}
    >
      Rename
    </Menu.Item>
  );
};

const splitMenuItemFactory = (
  direction: direction.Direction,
): FC<FocusMenuItemProps & { children?: ReactElement }> => {
  const C = ({
    layoutKey,
    children,
  }: FocusMenuItemProps & { children?: ReactElement }) => {
    const d = useDispatch();
    const [windowKey, mosaic] = useSelectMosaic();
    const canSplit = Mosaic.canSplit(mosaic, layoutKey);
    if (!canSplit) return null;
    return (
      <>
        {children}
        <Menu.Item
          itemKey={`split${direction}`}
          startIcon={direction === "x" ? <Icon.SplitX /> : <Icon.SplitY />}
          onClick={() =>
            d(splitMosaicNode({ windowKey, tabKey: layoutKey, direction }))
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
export const SplitXMenuItem = splitMenuItemFactory("x");
export const SplitYMenuItem = splitMenuItemFactory("y");

export interface MenuItems {
  layoutKey: string;
}

export const MenuItems = ({ layoutKey }: MenuItems): ReactElement => {
  return (
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
};
