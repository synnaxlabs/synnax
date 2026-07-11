// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/app/mosaic/Mosaic.css";

import { ontology } from "@synnaxlabs/client";
import { Logo } from "@synnaxlabs/media";
import {
  Breadcrumb,
  Button,
  Component,
  Dialog,
  Eraser,
  Errors,
  Flex,
  Flux,
  Icon,
  Menu,
  Mosaic as Base,
  Nav,
  type Pluto,
  Portal,
  Status,
  Synnax,
  Tabs,
  Text,
  Triggers,
} from "@synnaxlabs/pluto";
import { caseconv } from "@synnaxlabs/x";
import {
  memo,
  type ReactElement,
  type RefObject,
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
} from "react";

import { Nav as AppNav } from "@/app/nav";
import { Selector } from "@/app/selector";
import { LinePlot } from "@/feature/lineplot";
import { Log } from "@/feature/log";
import { Project } from "@/feature/project";
import { Schematic } from "@/feature/schematic";
import { Table } from "@/feature/table";
import { Task } from "@/feature/task";
import { ContextMenu as PlatformContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Import } from "@/platform/import";
import { Layout } from "@/platform/layout";
import { type Mosaic as PMosaic } from "@/platform/mosaic";
import { Session } from "@/session";

const EmptyContent = (): ReactElement => {
  const createComponentEnabled = Selector.useVisible();
  return (
    <Eraser.Eraser>
      <Flex.Box gap={5} center>
        <Logo className="synnax-logo-watermark" />
        {createComponentEnabled && (
          <Flex.Box x gap="small">
            <Text.Text level="h5" weight={450} color={9}>
              New Component
            </Text.Text>
            <Flex.Box x empty>
              <Triggers.Text level="h5" trigger={["Control", "T"]} />
            </Flex.Box>
          </Flex.Box>
        )}
      </Flex.Box>
    </Eraser.Eraser>
  );
};
export const LAYOUT_TYPE = "mosaic";

const ContextMenu = ({ keys }: Menu.ContextMenuMenuProps): ReactElement | null => {
  if (keys.length === 0)
    return (
      <PlatformContextMenu.Menu>
        <PlatformContextMenu.ReloadConsoleItem />
      </PlatformContextMenu.Menu>
    );
  const layoutKey = keys[0];
  const layout = Session.Layout.useSelect(layoutKey);
  if (layout == null) return null;
  const C = Layout.useContextMenuRenderer(layout.type);
  return C == null ? (
    <PlatformContextMenu.Menu>
      <Layout.MenuItems layoutKey={layoutKey} />
    </PlatformContextMenu.Menu>
  ) : (
    <C layoutKey={layoutKey} />
  );
};

interface ModalContentProps {
  tabKey: string;
  node: Portal.Node;
}

const ModalContent = ({ node, tabKey }: ModalContentProps): ReactElement => {
  const dispatch = Session.useDispatch();
  const layout = Session.Layout.useSelectRequired(tabKey);
  const { windowKey, focused: focusedKey } = Session.Layout.useSelectFocused();
  const focused = tabKey === focusedKey;
  const handleClose = () =>
    windowKey != null && dispatch(Session.Layout.setFocus({ windowKey, key: null }));
  const openInNewWindow = Layout.useOpenInNewWindow();
  const handleOpenInNewWindow = () => {
    openInNewWindow(tabKey);
    handleClose();
  };
  return (
    <Dialog.Frame
      onVisibleChange={handleClose}
      visible={focused}
      full
      modalPosition="slammed"
      variant="modal"
      background={focused ? 0 : undefined}
    >
      <Dialog.Dialog
        passthrough
        full
        className={CSS(CSS.B(caseconv.toKebab(layout.type)), CSS.B("mosaic-modal"))}
      >
        <Nav.Bar
          location="top"
          size="5rem"
          className={CSS(
            CSS.B("mosaic-modal-bar"),
            focused && CSS.BM("mosaic-modal-bar", "focused"),
          )}
          bordered
        >
          {/*
           * We do this to reduce the number of mounted DOM nodes. For some reason removing
           * the entire bar causes react to crash, so we just hide its children.
           */}
          {focused && (
            <>
              <Nav.Bar.Start>
                <Breadcrumb.Breadcrumb>
                  <Breadcrumb.Segment>
                    {Icon.resolve(layout.icon)}
                    {layout.name}
                  </Breadcrumb.Segment>
                </Breadcrumb.Breadcrumb>
              </Nav.Bar.Start>
              <Nav.Bar.End pack>
                {Session.Runtime.ENGINE === "tauri" && (
                  <Button.Button
                    onClick={handleOpenInNewWindow}
                    size="small"
                    textColor={9}
                  >
                    <Icon.OpenInNewWindow />
                  </Button.Button>
                )}
                <Button.Button onClick={handleClose} size="small" textColor={9}>
                  <Icon.Subtract />
                </Button.Button>
              </Nav.Bar.End>
            </>
          )}
        </Nav.Bar>
        <Portal.Out node={node} />
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};

const contextMenu = Component.renderProp(ContextMenu);

interface TabNameContentProps {
  tab: Session.Layout.Mosaic.Tab;
  /**
   * onCommit overrides the enclosing Tabs.Frame's rename wiring for names whose
   * rename flow involves more than the layout slice (e.g. ontology renames).
   */
  onCommit?: (name: string) => void;
}

const TabNameContent = ({ tab, onCommit }: TabNameContentProps): ReactElement => {
  const { tabKey, name, icon, loading, unsavedChanges, editable = true } = tab;
  const resolvedIcon = loading ? <Icon.Loading /> : (icon as Icon.ReactElement);
  let text: ReactElement;
  if (!editable)
    text = (
      <Text.Text level="p" overflow="ellipsis">
        {name}
      </Text.Text>
    );
  else if (onCommit != null)
    text = (
      <Text.Editable
        id={`pluto-tab-${tabKey}`}
        level="p"
        value={name}
        onChange={onCommit}
        overflow="ellipsis"
      />
    );
  else text = <Tabs.Name value={name} />;
  return (
    <>
      {resolvedIcon != null && Icon.resolve(resolvedIcon)}
      {text}
      {unsavedChanges === true && <Icon.Circle className="pluto-tabs__unsaved" />}
    </>
  );
};

interface CustomTabNameProps {
  useName: Layout.UseName;
  tab: Session.Layout.Mosaic.Tab;
  onRename: (key: string, name: string) => void;
}

const CustomTabName = ({
  useName,
  tab,
  onRename: propsOnRename,
}: CustomTabNameProps): ReactElement => {
  const { tabKey } = tab;
  const handleLayoutRename = useCallback(
    (name: string) => propsOnRename(tabKey, name),
    [tabKey, propsOnRename],
  );
  const { onRename, retrieve } = useName(tabKey, handleLayoutRename);
  useEffect(() => {
    retrieve();
  }, [retrieve]);
  const handleCommit = useCallback(
    (name: string) => {
      handleLayoutRename(name);
      onRename(name);
    },
    [handleLayoutRename, onRename],
  );
  return <TabNameContent tab={tab} onCommit={handleCommit} />;
};

interface TabNameProps {
  tab: Session.Layout.Mosaic.Tab;
  onRename: (key: string, name: string) => void;
}

const TabName = ({ tab, onRename }: TabNameProps): ReactElement => {
  const type = Session.Layout.useSelectType(tab.tabKey);
  const useName = Layout.useNameHook(type);
  if (useName != null)
    return <CustomTabName key={type} useName={useName} tab={tab} onRename={onRename} />;
  return <TabNameContent tab={tab} />;
};

interface LeafProps {
  node: Session.Layout.Mosaic.Node;
  activeTab: string | null;
  onSelect: (key: string) => void;
  onClose: (key: string) => void;
  onRename: (key: string, name: string) => void;
  onAdd?: (nodeKey: number) => void;
  portalRef: RefObject<Map<string, Portal.Node>>;
}

const Leaf = ({
  node,
  activeTab,
  onSelect,
  onClose,
  onRename,
  onAdd,
  portalRef,
}: LeafProps): ReactElement => {
  const { key, tabs = [], selected } = node;
  const menuProps = Menu.useContextMenu();
  const { startDrag, onDragEnd } = Base.useDragTab();
  const selectedNode = selected != null ? portalRef.current.get(selected) : undefined;
  return (
    <Base.Leaf leafKey={key.toString()} grow>
      <Tabs.Frame
        value={selected ?? ""}
        onChange={onSelect}
        onClose={onClose}
        onRename={onRename}
        grow
      >
        <Menu.ContextMenu
          style={{ height: "fit-content" }}
          {...menuProps}
          menu={contextMenu}
        />
        <Tabs.Selector
          className={menuProps.className}
          onContextMenu={menuProps.open}
          altColor={activeTab != null && activeTab === selected}
        >
          {tabs.map((tab) => (
            <Tabs.Tab
              key={tab.tabKey}
              itemKey={tab.tabKey}
              draggable
              onDragStart={(e) => startDrag(e, tab.tabKey)}
              onDragEnd={onDragEnd}
            >
              <TabName tab={tab} onRename={onRename} />
              {tab.closable !== false && <Tabs.Close />}
            </Tabs.Tab>
          ))}
          <Flex.Box grow />
          {onAdd != null && (
            <Button.Button
              variant="text"
              sharp
              onClick={() => onAdd(key)}
              tooltip="Create component"
            >
              <Icon.Add />
            </Button.Button>
          )}
        </Tabs.Selector>
        <Tabs.Content grow>
          {selected == null || selectedNode == null ? (
            <EmptyContent />
          ) : (
            <ModalContent key={selected} tabKey={selected} node={selectedNode} />
          )}
        </Tabs.Content>
      </Tabs.Frame>
    </Base.Leaf>
  );
};

interface MosaicProps {
  windowKey: string;
  mosaic: Session.Layout.Mosaic.Node;
}

export const Mosaic = memo((): ReactElement | null => {
  const [windowKey, mosaic] = Session.Layout.useSelectMosaic();
  return windowKey == null || mosaic == null ? null : (
    <Internal windowKey={windowKey} mosaic={mosaic} />
  );
});
Mosaic.displayName = "Mosaic";

const useMosaicDrops = (): Record<string, PMosaic.DropHandler> => ({
  schematic: Schematic.useMosaicDrop(),
  table: Table.useMosaicDrop(),
  lineplot: LinePlot.useMosaicDrop(),
  task: Task.useMosaicDrop(),
  log: Log.useMosaicDrop(),
});

const PORTAL_NODE_ATTRS = {
  style: "width: 100%; height: 100%; position: relative;",
};

/** LayoutMosaic renders the central layout mosaic of the application. */
const Internal = ({ windowKey, mosaic }: MosaicProps): ReactElement => {
  const store = Session.useStore();
  const activeTab = Session.Layout.useSelectActiveMosaicTabState();
  const client = Synnax.use();
  const placeLayout = Layout.usePlacer();
  const dispatch = Session.useDispatch();
  const handleError = Status.useErrorHandler();
  const fluxStore = Flux.useStore<Pluto.FluxStore>();
  const handleDrop = useCallback(
    ({ leafKey, tabKey, location, index }: Base.OnDropProps): void => {
      dispatch(
        Session.Layout.moveMosaicTab({
          key: Number(leafKey),
          tabKey,
          loc: location,
          windowKey,
          index,
        }),
      );
    },
    [dispatch, windowKey],
  );

  const mosaicDrops = useMosaicDrops();
  const fileIngesters = Import.useFileIngesters();

  const handleCreate = useCallback(
    ({ leafKey, location, tabKeys }: Base.OnCreateProps) => {
      const mosaicKey = Number(leafKey);
      tabKeys.forEach((tabKey) => {
        const res = ontology.idZ.safeParse(tabKey);
        if (res.success)
          mosaicDrops[res.data.type]?.({
            id: res.data,
            nodeKey: mosaicKey,
            location,
          });
        else placeLayout(Selector.create({ tab: { mosaicKey, location } }));
      });
    },
    [placeLayout, mosaicDrops],
  );

  const handleAdd = useCallback(
    (nodeKey: number) =>
      placeLayout(Selector.create({ tab: { mosaicKey: nodeKey, location: "center" } })),
    [placeLayout],
  );

  LinePlot.useTriggerHold();

  const handleClose = Layout.useRemover();

  const handleSelect = useCallback(
    (tabKey: string): void => {
      dispatch(Session.Layout.selectMosaicTab({ tabKey }));
    },
    [dispatch],
  );

  const handleRename = useCallback(
    (tabKey: string, name: string): void => {
      dispatch(Session.Layout.rename({ key: tabKey, name }));
    },
    [dispatch],
  );

  const handleResize = useCallback(
    (splitKey: string, size: number) => {
      dispatch(
        Session.Layout.resizeMosaicTab({ key: Number(splitKey), size, windowKey }),
      );
    },
    [dispatch, windowKey],
  );

  const handleFileDrop = useCallback(
    ({ leafKey, location, event }: Base.OnFileDropProps) => {
      const nodeKey = Number(leafKey);
      const items = Array.from(event.dataTransfer.items);
      void Promise.all(
        items.map(async (item) => {
          try {
            await Import.dataTransferItem(item, {
              client,
              fileIngesters,
              ingestDirectory: Project.ingest,
              layout: { tab: { mosaicKey: nodeKey, location } },
              placeLayout,
              store,
              fluxStore,
            });
          } catch (e) {
            handleError(e, `Failed to read ${item.getAsFile()?.name ?? "file"}`);
          }
        }),
      );
    },
    [client, fileIngesters, placeLayout, store, fluxStore, handleError],
  );

  // One traversal derives both portal enumeration inputs: every tab key in the
  // tree and the set of keys visible as their leaf's selection.
  const [tabKeys, visibleKeys] = useMemo(() => {
    const keys: string[] = [];
    const visible = new Set<string>();
    Session.Layout.Mosaic.forEachNode(mosaic, (node) => {
      node.tabs?.forEach((t) => keys.push(t.tabKey));
      if (node.selected != null) visible.add(node.selected);
    });
    return [keys, visible];
  }, [mosaic]);

  // Content renders into portal nodes hosted from each leaf, so moving layouts
  // around the mosaic or focusing them does not remount them. This has
  // considerable impact on the user experience, as it avoids re-fetching data
  // and re-creating expensive resources like WebGL contexts.
  const [portalRef, portalNodes] = Portal.useNodes({
    keys: tabKeys,
    attrs: PORTAL_NODE_ATTRS,
    onClick: handleSelect,
    children: (tabKey) => (
      <Errors.Boundary>
        <Layout.Content layoutKey={tabKey} forceHidden={!visibleKeys.has(tabKey)} />
      </Errors.Boundary>
    ),
  });

  const selectorVisible = Selector.useVisible();

  const renderNode = (node: Session.Layout.Mosaic.Node): ReactElement | null => {
    if (node.tabs != null)
      return (
        <Leaf
          key={node.key}
          node={node}
          activeTab={activeTab.layoutKey}
          onSelect={handleSelect}
          onClose={handleClose}
          onRename={handleRename}
          onAdd={selectorVisible ? handleAdd : undefined}
          portalRef={portalRef}
        />
      );
    if (node.first == null || node.last == null) {
      console.warn("mosaic tree is malformed");
      return null;
    }
    return (
      <Base.Split
        key={node.key}
        splitKey={node.key.toString()}
        id={`mosaic-${node.key}`}
        direction={node.direction}
        size={node.size}
      >
        {renderNode(node.first)}
        {renderNode(node.last)}
      </Base.Split>
    );
  };

  return (
    <>
      {portalNodes}
      <Base.Frame
        rounded={1}
        bordered
        borderColor={5}
        background={0}
        className={CSS.B("mosaic")}
        onDrop={handleDrop}
        onResize={handleResize}
        onCreate={selectorVisible ? handleCreate : undefined}
        onFileDrop={handleFileDrop}
      >
        {renderNode(mosaic)}
      </Base.Frame>
    </>
  );
};

export const Window = memo<Layout.Renderer>(({ layoutKey }: Layout.RendererProps) => {
  const dispatch = Session.useDispatch();
  const [windowKey, mosaic] = Session.Layout.useSelectMosaic();
  useLayoutEffect(() => {
    dispatch(Session.Nav.showBottom({}));
  }, [layoutKey]);
  if (windowKey == null || mosaic == null) return null;
  return (
    <>
      <AppNav.Bar.AuxTop />
      <Flex.Box
        y
        gap="tiny"
        grow
        className={CSS.B("mosaic-window")}
        style={{ padding: "1rem", paddingTop: 0, overflow: "hidden" }}
      >
        <Internal windowKey={windowKey} mosaic={mosaic} />
        <AppNav.Drawer.Bottom />
      </Flex.Box>
    </>
  );
});
Window.displayName = "MosaicWindow";

export const LAYOUTS: Layout.Renderers = {
  [LAYOUT_TYPE]: Mosaic,
  [Session.Layout.MOSAIC_WINDOW_TYPE]: Window,
};
