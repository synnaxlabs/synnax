// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layouts/Mosaic.css";

import { ontology } from "@synnaxlabs/client";
import { selectWindowKey } from "@synnaxlabs/drift";
import { Icon, Logo } from "@synnaxlabs/media";
import {
  Breadcrumb,
  Button,
  componentRenderProp,
  Eraser,
  Menu as PMenu,
  Modal,
  Mosaic as Core,
  Nav,
  OS,
  Portal,
  Status,
  Synnax,
  type Tabs,
  Text,
  useDebouncedCallback,
} from "@synnaxlabs/pluto";
import { type location } from "@synnaxlabs/x";
import { memo, type ReactElement, useCallback, useLayoutEffect } from "react";
import { useDispatch, useStore } from "react-redux";

import { Controls } from "@/components";
import { Menu } from "@/components/menu";
import { NAV_DRAWERS, NavDrawer, NavMenu } from "@/components/nav/Nav";
import { Confirm } from "@/confirm";
import { Layout } from "@/layout";
import { Content } from "@/layout/Content";
import { usePlacer } from "@/layout/hooks";
import { useMoveIntoMainWindow } from "@/layout/Menu";
import { useSelectActiveMosaicTabKey, useSelectMosaic } from "@/layout/selectors";
import {
  moveMosaicTab,
  remove,
  rename,
  resizeMosaicTab,
  selectMosaicTab,
  setNavDrawer,
} from "@/layout/slice";
import { createSelector } from "@/layouts/Selector";
import { LinePlot } from "@/lineplot";
import { Schematic } from "@/schematic";
import { SERVICES } from "@/services";
import { type RootState, type RootStore } from "@/store";
import { Workspace } from "@/workspace";

const EmptyContent = (): ReactElement => (
  <Eraser.Eraser>
    <Logo.Watermark />;
  </Eraser.Eraser>
);

const emptyContent = <EmptyContent />;

export const MOSAIC_TYPE = "mosaic";

const FILE_HANDLERS = [Schematic.fileHandler, LinePlot.fileHandler];

export const ContextMenu = ({
  keys,
}: PMenu.ContextMenuMenuProps): ReactElement | null => {
  if (keys.length === 0)
    return (
      <PMenu.Menu level="small" iconSpacing="small">
        <Menu.HardReloadItem />
      </PMenu.Menu>
    );
  const layoutKey = keys[0];
  const layout = Layout.useSelect(layoutKey);
  if (layout == null) return null;
  const C = Layout.useContextMenuRenderer(layout?.type);
  if (C == null)
    return (
      <PMenu.Menu level="small" iconSpacing="small">
        <Layout.MenuItems layoutKey={layoutKey} />
      </PMenu.Menu>
    );
  const res = <C layoutKey={layoutKey} />;
  return res;
};

interface ContentCProps extends Tabs.Tab {
  node: Portal.Node;
}

const ModalContent = ({ node, tabKey }: ContentCProps) => {
  const d = useDispatch();
  const layout = Layout.useSelectRequired(tabKey);
  const { windowKey, focused: focusedKey } = Layout.useSelectFocused();
  const focused = tabKey === focusedKey;
  const handleClose = () =>
    windowKey != null && d(Layout.setFocus({ windowKey, key: null }));
  const openInNewWindow = Layout.useOpenInNewWindow();
  const handleOpenInNewWindow = () => {
    openInNewWindow(tabKey);
    handleClose();
  };
  return (
    <Modal.Dialog visible close={handleClose} centered enabled={focused}>
      <Nav.Bar
        style={{ display: focused ? "flex" : "none" }}
        location="top"
        size="5rem"
      >
        <Nav.Bar.Start style={{ paddingLeft: "2rem" }}>
          <Breadcrumb.Breadcrumb icon={layout.icon}>
            {layout.name}
          </Breadcrumb.Breadcrumb>
        </Nav.Bar.Start>
        <Nav.Bar.End style={{ paddingRight: "1rem" }} empty>
          <Button.Icon onClick={handleOpenInNewWindow} size="small">
            <Icon.OpenInNewWindow style={{ color: "var(--pluto-gray-l8)" }} />
          </Button.Icon>
          <Button.Icon onClick={handleClose} size="small">
            <Icon.Subtract style={{ color: "var(--pluto-gray-l8)" }} />
          </Button.Icon>
        </Nav.Bar.End>
      </Nav.Bar>
      <Portal.Out node={node} />
    </Modal.Dialog>
  );
};

const contextMenu = componentRenderProp(ContextMenu);

/** LayoutMosaic renders the central layout mosaic of the application. */
export const Mosaic = memo((): ReactElement => {
  const [windowKey, mosaic] = useSelectMosaic();
  const store = useStore();
  const activeTab = useSelectActiveMosaicTabKey();
  const client = Synnax.use();
  const placer = usePlacer();
  const dispatch = useDispatch();
  const addStatus = Status.useAggregator();

  const handleDrop = useCallback(
    (key: number, tabKey: string, loc: location.Location): void => {
      dispatch(moveMosaicTab({ key, tabKey, loc, windowKey }));
    },
    [dispatch, windowKey],
  );

  const handleCreate = useCallback(
    (mosaicKey: number, location: location.Location, tabKeys?: string[]) => {
      if (tabKeys == null) {
        placer(
          createSelector({
            tab: { mosaicKey, location },
            location: "mosaic",
          }),
        );
        return;
      }
      tabKeys.forEach((tabKey) => {
        const res = ontology.stringIDZ.safeParse(tabKey);
        if (res.success) {
          const id = new ontology.ID(res.data);
          if (client == null) return;
          SERVICES[id.type].onMosaicDrop?.({
            client,
            store: store as RootStore,
            id,
            nodeKey: mosaicKey,
            location,
            placeLayout: placer,
            addStatus,
          });
        } else
          placer(
            createSelector({
              tab: { mosaicKey, location },
              location: "mosaic",
            }),
          );
      });
    },
    [placer, store, client, addStatus],
  );

  LinePlot.useTriggerHold({
    defaultMode: "hold",
    hold: [["H"]],
    toggle: [["H", "H"]],
  });

  const handleClose = useCallback(
    (tabKey: string): void => {
      dispatch(remove({ keys: [tabKey] }));
    },
    [dispatch],
  );

  const handleSelect = useCallback(
    (tabKey: string): void => {
      dispatch(selectMosaicTab({ tabKey }));
    },
    [dispatch],
  );

  const handleRename = useCallback(
    (tabKey: string, name: string): void => {
      dispatch(rename({ key: tabKey, name }));
    },
    [dispatch],
  );

  const handleResize = useDebouncedCallback(
    (key, size) => {
      dispatch(resizeMosaicTab({ key, size, windowKey }));
    },
    100,
    [dispatch, windowKey],
  );

  const workspaceKey = Workspace.useSelectActiveKey();
  const confirm = Confirm.useModal();

  const handleFileDrop = useCallback(
    (nodeKey: number, loc: location.Location, event: React.DragEvent) => {
      void (async () => {
        const files = Array.from(event.dataTransfer.files);
        for (const file of files) {
          const name = file.name;
          try {
            if (file.type !== "application/json")
              throw Error(`${name} is not a JSON file`);
            const buffer = await file.arrayBuffer();
            const fileAsJSON = JSON.parse(new TextDecoder().decode(buffer));

            let handlerFound = false;
            for (const fileHandler of FILE_HANDLERS)
              if (
                await fileHandler({
                  file: fileAsJSON,
                  placer,
                  name,
                  store,
                  confirm,
                  client,
                  workspaceKey: workspaceKey ?? undefined,
                  dispatch,
                  tab: { mosaicKey: nodeKey, location: loc },
                })
              ) {
                handlerFound = true;
                break;
              }
            if (!handlerFound)
              throw Error(`${name} is not recognized as a Synnax object`);
          } catch (e) {
            if (e instanceof Error)
              addStatus({
                variant: "error",
                message: `Failed to read ${name}`,
                description: e.message,
              });
          }
        }
      })();
    },
    [dispatch],
  );

  // Creates a wrapper around the general purpose layout content to create a set of
  // content nodes that are rendered at the top level of the Mosaic and then 'portaled'
  // into their correct location. This means that moving layouts around in the Mosaic
  // or focusing them will not cause them to re-mount. This has considerable impacts
  // on the user experience, as it reduces necessary data fetching and expensive
  const [portalRef, portalNodes] = Core.usePortal({
    root: mosaic,
    onSelect: handleSelect,
    children: ({ tabKey, visible }) => (
      <Content key={tabKey} layoutKey={tabKey} forceHidden={visible === false} />
    ),
  });

  const renderProp = useCallback<Tabs.RenderProp>(
    (props) => (
      <ModalContent
        key={props.tabKey}
        node={portalRef.current.get(props.tabKey) as Portal.Node}
        {...props}
      />
    ),
    [],
  );

  return (
    <>
      {portalNodes}
      <Core.Mosaic
        root={mosaic}
        onDrop={handleDrop}
        onClose={handleClose}
        onSelect={handleSelect}
        contextMenu={contextMenu}
        onResize={handleResize}
        emptyContent={emptyContent}
        onRename={handleRename}
        onCreate={handleCreate}
        activeTab={activeTab ?? undefined}
        onFileDrop={handleFileDrop}
      >
        {renderProp}
      </Core.Mosaic>
    </>
  );
});
Mosaic.displayName = "Mosaic";

export const NavTop = (): ReactElement | null => {
  const os = OS.use();
  const active = Layout.useSelectActiveMosaicLayout();
  const ws = Workspace.useSelectActive();
  const store = useStore<RootState>();
  const moveToMain = useMoveIntoMainWindow();
  const collapseButton = (
    <Button.Icon
      onClick={() => {
        const state = store.getState();
        const winKey = selectWindowKey(state);
        Object.values(state.layout.layouts)
          .filter((l) => l.windowKey === winKey && l.location === "mosaic")
          .forEach((l) => moveToMain(l.key));
      }}
    >
      <Icon.MoveToMainWindow />
    </Button.Icon>
  );
  return (
    <Nav.Bar
      className="console-main-nav-top"
      location="top"
      size={"5rem"}
      data-tauri-drag-region
    >
      <Nav.Bar.Start className="console-main-nav-top__start" data-tauri-drag-region>
        <Controls
          className="console-controls--macos"
          visibleIfOS="MacOS"
          forceOS={os}
        />
        {os === "Windows" && <Logo className="console-main-nav-top__logo" />}
      </Nav.Bar.Start>
      <Nav.Bar.AbsoluteCenter data-tauri-drag-region>
        <Text.Text
          level="p"
          shade={6}
          style={{ transform: "scale(0.9)", cursor: "default" }}
          data-tauri-drag-region
        >
          {active?.name} {ws?.name != null ? ` - ${ws.name}` : ""}
        </Text.Text>
      </Nav.Bar.AbsoluteCenter>
      <Nav.Bar.End
        data-tauri-drag-region
        style={{ paddingRight: os == "Windows" ? "0" : "1.5rem" }}
      >
        {collapseButton}
        {os === "Windows" && (
          <Controls
            className="console-controls--windows"
            visibleIfOS="Windows"
            forceOS={os}
          />
        )}
      </Nav.Bar.End>
    </Nav.Bar>
  );
};

export const MosaicWindow = memo(
  ({ layoutKey }: Layout.RendererProps): ReactElement => {
    const { menuItems, onSelect } = Layout.useNavDrawer("bottom", NAV_DRAWERS);
    const dispatch = useDispatch();
    useLayoutEffect(() => {
      dispatch(
        setNavDrawer({
          windowKey: layoutKey,
          location: "bottom",
          menuItems: ["visualization"],
          activeItem: "visualization",
        }),
      );
    }, [layoutKey]);
    return (
      <>
        <NavTop />
        <Mosaic />
        <NavDrawer location="bottom" />
        <Nav.Bar
          className="console-main-nav"
          location="bottom"
          style={{ paddingRight: "1.5rem", zIndex: 8 }}
          size="6rem"
        >
          <Nav.Bar.End>
            <NavMenu onChange={onSelect}>{menuItems}</NavMenu>
          </Nav.Bar.End>
        </Nav.Bar>
      </>
    );
  },
);
MosaicWindow.displayName = "MosaicWindow";
