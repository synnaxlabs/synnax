// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/panel/Mosaic.css";

import { panel, query } from "@synnaxlabs/client";
import { type direction } from "@synnaxlabs/x";
import {
  type DragEventHandler,
  memo,
  type ReactElement,
  type ReactNode,
  useCallback,
} from "react";

import { Button } from "@/button";
import { type Component } from "@/component";
import { CSS } from "@/css";
import { Errors } from "@/errors";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Menu } from "@/menu";
import { Mosaic as Base } from "@/mosaic";
import { createTabDragPayload, parseTabDragPayload } from "@/panel/haul";
import {
  useCanEdit,
  useLeafNode,
  useMoveTabToPanel,
  useNodeVariant,
  useRoot,
  useSingleDispatch,
  useSplitNode,
  useTabKeys,
} from "@/panel/queries";
import { Scope, TabScope } from "@/panel/scope";
import { Portal } from "@/portal";
import { Select } from "@/select";
import { Status } from "@/status/base";
import { Synnax } from "@/synnax";
import { Tabs } from "@/tabs";
import { Triggers } from "@/triggers";

export interface MosaicProps extends Omit<
  Base.FrameProps,
  "onDrop" | "onCreate" | "onResize" | "onSelect" | "children" | "contextMenu"
> {
  selected?: panel.TabKey[];
  onSelect?: (tabKey: panel.TabKey) => void;
  children: Component.RenderProp<{}>;
  tabName?: Component.RenderProp<{}>;
  onCreateTab?: () => panel.NewTab | undefined;
  resolveDroppedTab?: (key: string) => panel.NewTab | undefined;
  /** Renders the full tab context menu. When a tab is under the cursor, the
   * render prop runs inside that tab's scope, so {@link CloseTabMenuItem} and
   * {@link SplitTabMenuItems} can be composed with caller items in any order. */
  contextMenu?: Component.RenderProp<Menu.ContextMenuMenuProps>;
  /** Rendered in a leaf's content area when the leaf has no tabs. */
  emptyContent?: ReactNode;
  /** Tab the mosaic collapses to: the frame renders a single leaf holding this
   * tab in place of the tree. Content stays mounted through the portal layer;
   * background tabs detach from the DOM, so their canvas draws stop. */
  overlaid?: panel.TabKey;
  /** Called when the user requests to exit the overlaid state. */
  onStopOverlay?: () => void;
}

interface TabProps extends Pick<MosaicProps, "tabName"> {
  tabKey: panel.TabKey;
  onClose: (tabKey: panel.TabKey) => void;
  canEdit: boolean;
}

const Tab = ({ tabKey, tabName, onClose, canEdit }: TabProps): ReactElement => {
  const { startDrag, onDragEnd } = Base.useDragTab();
  const key = Scope.use();
  const client = Synnax.use();
  const handleDragStart = useCallback<DragEventHandler<HTMLDivElement>>(
    (e) => {
      const cached = client?.panels.getCached(key);
      if (!query.isLive(cached)) return;
      const tab = panel.findTab(cached.root, tabKey);
      if (tab == null) return;
      startDrag(e, tabKey, createTabDragPayload(key, tab));
    },
    [tabKey, startDrag, key, client],
  );
  const handleClose = useCallback(() => onClose(tabKey), [tabKey, onClose]);
  return (
    <Tabs.Tab
      key={tabKey}
      itemKey={tabKey}
      draggable={canEdit}
      onDragStart={handleDragStart}
      onDragEnd={onDragEnd}
      // Tabs.Tab binds Delete to onClose, so withholding it also closes that route.
      onClose={canEdit ? handleClose : undefined}
    >
      <TabScope.Provider value={tabKey}>{tabName?.({})}</TabScope.Provider>
      {canEdit && <Tabs.Close onClick={handleClose} />}
    </Tabs.Tab>
  );
};

interface NodeProps
  extends
    Pick<Base.LeafProps, "nodeKey">,
    Pick<TabProps, "tabName" | "onClose" | "canEdit"> {
  onAdd: (nodeKey: number) => void;
  onContextMenu: Menu.ContextMenuOpen;
  emptyContent?: ReactNode;
}

/** Creates a new tab. Bound by the embedding app; shown on the create button. */
export const CREATE_TAB_TRIGGER: Triggers.Trigger = ["Control", "T"];

const CREATE_TAB_TOOLTIP = <Triggers.Text trigger={CREATE_TAB_TRIGGER} level="small" />;

const EXIT_FOCUS_TOOLTIP = <Triggers.Text trigger={Triggers.ESCAPE} level="small" />;

interface CreateTabProps {
  /** The leaf's own selected tab, absent when the leaf holds none. */
  selected?: panel.TabKey;
  onAdd: () => void;
}

/**
 * The button that adds a tab to a leaf. Split from the leaf so the subscription to
 * the selection head re-renders the button alone: focus moving between leaves must
 * not re-render either leaf's tabs.
 */
const CreateTab = memo(({ selected, onAdd }: CreateTabProps): ReactElement => {
  // Create acts on the panel's focused tab, so only the leaf holding that tab may
  // advertise the shortcut. An empty leaf holds none, and "" matches no tab.
  const { head } = Select.useItemState(selected ?? "");
  return (
    <Button.Button
      variant="text"
      size="small"
      onClick={onAdd}
      className={CSS.BE("panel-mosaic", "create")}
      tooltip={head ? CREATE_TAB_TOOLTIP : undefined}
    >
      <Icon.Add color={9} />
    </Button.Button>
  );
});
CreateTab.displayName = "Panel.Mosaic.CreateTab";

const Leaf = memo(
  ({
    nodeKey,
    onAdd,
    onContextMenu,
    emptyContent,
    canEdit,
    ...rest
  }: NodeProps): ReactElement => {
    const { tabs } = useLeafNode({ nodeKey });
    const selected = Select.useSelectedAmong(tabs) ?? tabs[0];
    const { onSelect } = Select.useContext();
    const handleAdd = useCallback(() => onAdd(nodeKey), [nodeKey, onAdd]);
    const handleSelectContent = useCallback(
      () => onSelect(selected),
      [onSelect, selected],
    );
    const selectorDropProps = Base.useSelectorDropProps({ nodeKey, tabKeys: tabs });
    return (
      <Base.Leaf nodeKey={nodeKey} grow>
        <Tabs.Frame grow>
          <Tabs.Selector
            align="start"
            {...selectorDropProps}
            onContextMenu={onContextMenu}
          >
            {tabs.map((tabKey) => (
              <Tab key={tabKey} tabKey={tabKey} canEdit={canEdit} {...rest} />
            ))}
            <Flex.Box grow />
            {canEdit && (
              <Flex.Box
                align="center"
                justify="center"
                className={CSS.BE("panel-mosaic", "cap")}
              >
                <CreateTab selected={selected} onAdd={handleAdd} />
              </Flex.Box>
            )}
          </Tabs.Selector>
          <Tabs.Content grow>
            {selected != null ? (
              <Portal.Out
                itemKey={selected}
                className={CSS.BE("panel-mosaic", "portal-out")}
                onClickCapture={handleSelectContent}
              />
            ) : (
              emptyContent
            )}
            <Base.Shield />
          </Tabs.Content>
        </Tabs.Frame>
      </Base.Leaf>
    );
  },
);
Leaf.displayName = "Panel.Mosaic.Leaf";

const Split = memo(({ nodeKey, ...rest }: NodeProps): ReactElement => {
  const { direction, size } = useSplitNode({ nodeKey });
  return (
    <Base.Split nodeKey={nodeKey} direction={direction} size={size}>
      <Node nodeKey={panel.childNodeKey(nodeKey, "first")} {...rest} />
      <Node nodeKey={panel.childNodeKey(nodeKey, "last")} {...rest} />
    </Base.Split>
  );
});
Split.displayName = "Panel.Mosaic.Split";

const Node = memo(({ nodeKey, ...rest }: NodeProps): ReactElement => {
  const C = useNodeVariant({ nodeKey }) == "split" ? Split : Leaf;
  return <C nodeKey={nodeKey} {...rest} />;
});
Node.displayName = "Panel.Mosaic.Node";

/** Enters a tab's overlaid (focused) state. Escape is the way out, so the exit
 * affordances hint that instead. Bound by the embedding app. */
export const OVERLAY_TRIGGER: Triggers.Trigger = ["Control", "L"];

interface OverlaidLeafProps extends Pick<TabProps, "tabName" | "onClose" | "canEdit"> {
  overlaid: panel.TabKey;
  onStopOverlay?: () => void;
  onContextMenu: Menu.ContextMenuOpen;
}

const OverlaidLeaf = ({
  overlaid,
  onStopOverlay,
  onContextMenu,
  ...rest
}: OverlaidLeafProps): ReactElement => (
  <Tabs.Frame grow className={CSS.BE("panel-mosaic", "overlaid-leaf")}>
    <Tabs.Selector align="start" onContextMenu={onContextMenu}>
      <Tab tabKey={overlaid} {...rest} />
      <Flex.Box grow />
      <Button.Button
        variant="text"
        size="small"
        onClick={onStopOverlay}
        className={CSS.BE("panel-mosaic", "overlaid-exit")}
        tooltip={EXIT_FOCUS_TOOLTIP}
      >
        <Icon.Collapse />
        Exit focus
      </Button.Button>
    </Tabs.Selector>
    <Tabs.Content grow>
      <Portal.Out itemKey={overlaid} className={CSS.BE("panel-mosaic", "portal-out")} />
    </Tabs.Content>
  </Tabs.Frame>
);

/** Closes the focused tab. Bound by the embedding app; shown on the close menu item. */
export const CLOSE_TRIGGER: Triggers.Trigger = ["Control", "W"];

/** Props for {@link CloseTabMenuItem}. */
export interface CloseTabMenuItemProps {
  /** Shows the {@link CLOSE_TRIGGER} hint. The app binds the trigger and chooses the
   * tab it acts on, so only the app knows whether this tab is that tab. */
  triggerIndicator?: boolean;
}

/** CloseTabMenuItem closes the context menu's tab. Hidden from a viewer who cannot
 * write the panel. Must render inside the tab context menu passed to {@link Mosaic}. */
export const CloseTabMenuItem = ({
  triggerIndicator = false,
}: CloseTabMenuItemProps = {}): ReactElement | null => {
  const tabKey = TabScope.use();
  const dispatch = useSingleDispatch();
  const canEdit = useCanEdit({});
  const handleClose = useCallback(
    () => dispatch(panel.removeTab({ key: tabKey })),
    [dispatch, tabKey],
  );
  if (!canEdit) return null;
  return (
    <Menu.Item
      itemKey="close"
      onClick={handleClose}
      triggerIndicator={triggerIndicator && CLOSE_TRIGGER}
    >
      <Icon.Close />
      Close
    </Menu.Item>
  );
};

/** SplitTabMenuItems splits the context menu's tab horizontally or vertically.
 * Hidden when the tab cannot be split, or from a viewer who cannot write the panel.
 * Must render inside the tab context menu passed to {@link Mosaic}. */
export const SplitTabMenuItems = (): ReactElement | null => {
  const tabKey = TabScope.use();
  const dispatch = useSingleDispatch();
  const root = useRoot({});
  const canEdit = useCanEdit({});
  const handleSplit = useCallback(
    (direction: direction.Direction) =>
      dispatch(panel.splitTab({ key: tabKey, direction })),
    [dispatch, tabKey],
  );
  if (!canEdit || !panel.canSplitTab(root, tabKey)) return null;
  return (
    <>
      <Menu.Item itemKey="splitX" onClick={() => handleSplit("x")}>
        <Icon.SplitX />
        Split horizontally
      </Menu.Item>
      <Menu.Item itemKey="splitY" onClick={() => handleSplit("y")}>
        <Icon.SplitY />
        Split vertically
      </Menu.Item>
    </>
  );
};

const EMPTY_SELECTED: panel.TabKey[] = [];

const PortalIn = memo(
  ({
    itemKey,
    children,
  }: Pick<Portal.InProps, "itemKey"> & Pick<MosaicProps, "children">): ReactElement => (
    <Portal.In itemKey={itemKey}>
      <Errors.Boundary>
        <TabScope.Provider value={itemKey}>{children({})}</TabScope.Provider>
      </Errors.Boundary>
    </Portal.In>
  ),
);
PortalIn.displayName = "Panel.Mosaic.PortalIn";

// Content renders into portaled elements hosted from each leaf, so moving a tab around
// the mosaic does not remount it: the destination leaf's Out re-parents the same
// element, preserving DOM state and expensive resources like WebGL contexts.
const PortaledContents = memo(
  ({ children }: Pick<MosaicProps, "children">): ReactElement => {
    const keys = useTabKeys();
    return (
      <>
        {keys.map((key) => (
          <PortalIn key={key} itemKey={key}>
            {children}
          </PortalIn>
        ))}
      </>
    );
  },
);
PortaledContents.displayName = "Panel.Mosaic.PortaledContents";

export const Mosaic = ({
  selected = EMPTY_SELECTED,
  onSelect,
  children,
  tabName,
  onCreateTab,
  resolveDroppedTab,
  contextMenu,
  emptyContent,
  overlaid,
  onStopOverlay,
  className,
  ...rest
}: MosaicProps): ReactElement | null => {
  const dispatch = useSingleDispatch();
  const key = Scope.use();
  const client = Synnax.use();
  const canEdit = useCanEdit({});

  const moveToPanel = useMoveTabToPanel();
  const handleError = Status.useErrorHandler();

  const handleDrop = useCallback(
    ({ nodeKey, tabKey, location, index, data }: Base.OnDropProps) => {
      const source = parseTabDragPayload(data);
      if (source == null || source.panel === key) {
        dispatch(panel.moveTab({ key: tabKey, targetLeaf: nodeKey, index, location }));
        // Another client can close the tab mid-drag, leaving the move a no-op and
        // the key naming a tab this panel no longer holds.
        const cached = client?.panels.getCached(key);
        if (query.isLive(cached) && panel.findTab(cached.root, tabKey) != null)
          onSelect?.(tabKey);
        return;
      }
      handleError(async () => {
        const landed = await moveToPanel({
          source: source.panel,
          destination: key,
          tab: source.tab,
          targetLeaf: nodeKey,
          index,
          location,
        });
        if (landed != null) onSelect?.(landed);
      }, "Failed to move tab");
    },
    [client, dispatch, moveToPanel, key, onSelect, handleError],
  );

  const handleResize = useCallback(
    (split: number, size: number) => dispatch(panel.resizeSplit({ split, size })),
    [dispatch],
  );

  const handleClose = useCallback(
    (key: string) => dispatch(panel.removeTab({ key })),
    [dispatch],
  );

  const handleAdd = useCallback(
    (path: number) => {
      const tab = onCreateTab?.();
      if (tab == null) return;
      const action = panel.insertTabs({ tabs: [tab], targetLeaf: path });
      dispatch(action);
      const inserted =
        action.type === "insert_tabs" ? action.insertTabs.tabs[0] : undefined;
      if (inserted != null) onSelect?.(inserted.key);
    },
    [dispatch, onSelect, onCreateTab],
  );

  const handleCreate = useCallback(
    ({ nodeKey, location, tabKeys, index }: Base.OnCreateProps) => {
      const tabs = tabKeys
        .map((tabKey) => resolveDroppedTab?.(tabKey))
        .filter((tab): tab is panel.NewTab => tab != null);
      if (tabs.length === 0) return;
      const action = panel.insertTabs({ tabs, targetLeaf: nodeKey, location, index });
      dispatch(action);
      const last =
        action.type === "insert_tabs" ? action.insertTabs.tabs.at(-1) : undefined;
      if (last != null) onSelect?.(last.key);
    },
    [dispatch, onSelect, resolveDroppedTab],
  );

  const menuProps = Menu.useContextMenu();
  const renderMenu = useCallback<Component.RenderProp<Menu.ContextMenuMenuProps>>(
    (props) => {
      const tabKey: panel.TabKey | undefined = props.keys[0];
      const content = (
        <Menu.Menu level="small" gap="small">
          {contextMenu?.(props)}
        </Menu.Menu>
      );
      if (tabKey == null) return content;
      return <TabScope.Provider value={tabKey}>{content}</TabScope.Provider>;
    },
    [contextMenu],
  );

  return (
    <Portal.Context>
      <PortaledContents>{children}</PortaledContents>
      <Select.Context value={selected} onSelect={onSelect}>
        <Menu.ContextMenu menu={renderMenu} {...menuProps}>
          <Base.Frame
            onDrop={handleDrop}
            onCreate={handleCreate}
            onResize={handleResize}
            disabled={!canEdit}
            className={CSS.cls(
              className,
              overlaid != null && CSS.BM("panel-mosaic", "overlaid"),
            )}
            {...rest}
          >
            {overlaid == null ? (
              <Node
                nodeKey={panel.ROOT_NODE_KEY}
                onClose={handleClose}
                onAdd={handleAdd}
                onContextMenu={menuProps.open}
                tabName={tabName}
                emptyContent={emptyContent}
                canEdit={canEdit}
              />
            ) : (
              <OverlaidLeaf
                overlaid={overlaid}
                tabName={tabName}
                onClose={handleClose}
                onStopOverlay={onStopOverlay}
                onContextMenu={menuProps.open}
                canEdit={canEdit}
              />
            )}
          </Base.Frame>
        </Menu.ContextMenu>
      </Select.Context>
    </Portal.Context>
  );
};
