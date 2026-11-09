// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, resource, type Synnax as Client } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { context } from "@synnaxlabs/lyra/context";
import { Haul } from "@synnaxlabs/lyra/haul";
import { useCombinedStateAndRef, useInitializerRef } from "@synnaxlabs/lyra/hooks";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Tree as Base } from "@synnaxlabs/lyra/tree";
import { Access, Synnax } from "@synnaxlabs/pluto";
import { array, state as xstate, unique } from "@synnaxlabs/x";
import {
  type ClipboardEvent,
  type FC,
  type FocusEvent,
  type KeyboardEvent,
  type ReactElement,
  type ReactNode,
  useEffect,
  useState,
  useSyncExternalStore,
} from "react";

import { CSS } from "@/platform/css";
import { type Source } from "@/platform/tree/children";
import { LIMIT } from "@/platform/tree/createSearch";
import {
  DefaultContextMenu,
  type DefaultContextMenuProps,
} from "@/platform/tree/DefaultContextMenu";
import { type DragAndDrop, useDragAndDrop } from "@/platform/tree/dragAndDrop";
import { DefaultItem, type Item } from "@/platform/tree/item";
import { useMenu } from "@/platform/tree/menu";
import { useItems } from "@/platform/tree/Provider";
import {
  MatchProvider,
  type ResultNodes,
  type Search,
  type SearchAnswer,
} from "@/platform/tree/search";
import { isMoreKey, Store } from "@/platform/tree/store";
import { type GetResource, type Resource, type TreeState } from "@/platform/tree/types";
import { useCopy } from "@/platform/tree/useCopy";

interface InternalProps extends Pick<List.ScrollProps, "ref" | "onKeyDown"> {
  client: Client;
  root: resource.ID;
  /**
   * Lists the children of the root. Defaults to the children the item of the root's
   * type lists, for a root that is a resource.
   */
  listRoot?: (client: Client) => Source[];
  emptyContent?: ReactNode;
  /** The menu for a right-click on empty space. */
  rootContextMenu?: FC<DefaultContextMenuProps>;
  results?: ResultNodes;
  className?: string;
}

interface ContextValue extends DragAndDrop {
  store: Store;
}

const [Context, useContext] = context.create<ContextValue>({
  displayName: "Tree.Context",
  providerName: "Tree.Tree",
});

// Loads the next page of its folder on a click, or on Enter, which double-clicks it.
const MoreItem = ({ onDrop: _, ...rest }: Base.ItemProps<string>) => {
  const { itemKey } = rest;
  const { store } = useContext("Tree.MoreItem");
  const loading = useSyncExternalStore(
    (handler) => store.subscribeKey(handler, itemKey),
    () => store.isLoading(itemKey),
  );
  const handleMore = () => store.more(itemKey);
  return (
    <Base.Item
      {...rest}
      onClick={handleMore}
      onDoubleClick={handleMore}
      loading={loading}
    >
      <Text.Text level="small" color={9}>
        Show more
      </Text.Text>
    </Base.Item>
  );
};

const ResourceItem = ({ onDrop: _, ...rest }: Base.ItemProps<string>) => {
  const { itemKey } = rest;
  const id = resource.parseID(itemKey);
  const treeResource = List.useItem<string, Resource>(itemKey);
  const Item = useItems()[id.type] ?? DefaultItem;
  const { store, onDrop, onDragStart, onDragEnd } = useContext("Tree.itemRenderProp");
  const loading = useSyncExternalStore(
    (handler) => store.subscribeKey(handler, itemKey),
    () => store.isLoading(itemKey),
  );
  // A drop re-parents the dragged items, so the grant is read on them and not on the
  // destination. The item itself carries the same grant to leave its parent.
  const client = Synnax.use();
  const canMove = Access.useUpdateGranted(id);

  const [draggingOver, setDraggingOver] = useState(false);

  const onDropDrops = Haul.useDrop({
    type: Base.HAUL_TYPE,
    key: itemKey,
    canDrop: ({ items: entities, source }) => {
      const keys = entities.map((item) => item.key);
      setDraggingOver(false);
      if (source.type !== Base.HAUL_TYPE || keys.includes(itemKey)) return false;
      return Access.updateGranted({
        client,
        id: Base.filterHaulItems(entities).map(({ key }) => resource.idZ.parse(key)),
      });
    },
    onDrop: (props) => onDrop(itemKey, props),
    onDragOver: () => setDraggingOver(true),
  });

  // The virtualizer has already reserved this row's slot. Rendering nothing would
  // leave a gap, so the row stays empty until the resource loads.
  if (treeResource == null) return <Base.Item {...rest} />;

  return (
    <Item
      {...rest}
      draggingOver={draggingOver}
      onDragStart={() => onDragStart(itemKey)}
      draggable={canMove}
      {...onDropDrops}
      onDragLeave={() => setDraggingOver(false)}
      onDragEnd={onDragEnd}
      resource={treeResource}
      loading={loading}
    />
  );
};

const TreeItem = (props: Base.ItemProps<string>) =>
  isMoreKey(props.itemKey) ? <MoreItem {...props} /> : <ResourceItem {...props} />;

const itemRenderProp = Component.renderProp(TreeItem);

interface CreateStoreParams extends Pick<
  InternalProps,
  "client" | "root" | "listRoot" | "results"
> {
  resolveItem: (type: resource.Type) => Item;
  handleError: Status.ErrorHandler;
}

const createStore = ({
  client,
  root,
  listRoot,
  results,
  resolveItem,
  handleError,
}: CreateStoreParams): Store =>
  new Store({
    root,
    list: (id) => {
      if (listRoot != null && resource.idsEqual(id, root)) return listRoot(client);
      const { children } = resolveItem(id.type);
      if (children == null) throw new Error(`no children resolver for ${id.type}`);
      return children({ client, id, root });
    },
    hasChildren: (type) => resolveItem(type).hasChildren,
    onError: (e) => handleError(e, "Failed to retrieve resources"),
    results,
  });

const Internal = ({
  client,
  root,
  listRoot,
  emptyContent,
  rootContextMenu = DefaultContextMenu,
  results,
  className,
  ref,
  onKeyDown,
}: InternalProps): ReactElement => {
  const items = useItems();
  const resolveItem = (type: resource.Type): Item => items[type] ?? DefaultItem;
  const handleError = Status.useErrorHandler();
  const rootKey = resource.idToString(root);
  const create = () =>
    createStore({ client, root, listRoot, results, resolveItem, handleError });
  // The store reads cached lists as it is made, so the first render paints them.
  const [held, setHeld] = useState(() => ({ client, store: create() }));
  let { store } = held;
  if (held.client !== client) {
    store = create();
    setHeld({ client, store });
  }
  const nodes = useSyncExternalStore(store.subscribe, store.getNodes);
  const expanded = useSyncExternalStore(store.subscribe, store.getExpanded);
  const [selected, setSelected, selectedRef] = useCombinedStateAndRef<string[]>([]);

  const handleExpandedChange = (arg: xstate.SetArg<string[]>) =>
    store.setExpanded(xstate.executeSetter(arg, store.getExpanded()));

  useEffect(() => store.start(), [store]);

  // Groups open on arrival once, so a group the user closes stays closed.
  const openedRef = useInitializerRef(() => new Set<string>());
  useEffect(() => {
    if (results == null) return;
    store.setResults(results);
    const opening = results.expanded.filter((key) => !openedRef.current.has(key));
    opening.forEach((key) => openedRef.current.add(key));
    if (opening.length > 0)
      handleExpandedChange((prev) => unique.unique([...prev, ...opening]));
  }, [store, results]);

  // Selection drives the context menu, and a key with no node poisons every later
  // right-click, so a node that leaves the tree leaves the selection with it.
  useEffect(() => {
    setSelected((prev) => {
      const next = prev.filter((key) => store.getParent(key) != null);
      return next.length === prev.length ? prev : next;
    });
  }, [nodes, setSelected]);

  const treeProps = Base.use({
    nodes,
    selected,
    onSelectedChange: setSelected,
    expanded,
    onExpandedChange: handleExpandedChange,
  });
  const { shape, expand, contract } = treeProps;

  const getResource = ((id: resource.ID | resource.ID[] | string | string[]) => {
    const ids = array.toArray(id);
    const resources = resource
      .idToString(ids)
      .map(store.get)
      .filter((r) => r != null);
    if (Array.isArray(id)) return resources;
    if (resources[0] == null)
      throw new NotFoundError(`Resource ${resource.idToString(id)} not found`);
    return resources[0];
  }) as GetResource;

  const getItem = List.createGetItem<string, Resource>(store.get, (keys) =>
    keys.map(store.get).filter((r) => r != null),
  );

  const state: TreeState = {
    shape,
    getChildren: (key) => store.getChildren(key),
    getParent: (key) => store.getParent(key),
    addDraft: (draft) => store.addDraft(draft),
    getResource,
    subscribeResource: store.subscribeKey,
    setSelection: setSelected,
    expand,
    contract,
  };

  const dragAndDrop = useDragAndDrop({
    client,
    store,
    root,
    results,
    resolveItem,
    selectedRef,
    contract,
    getResource,
    handleError,
  });
  const renderMenu = useMenu({
    client,
    store,
    root,
    results,
    selected,
    state,
    rootContextMenu,
    resolveItem,
    handleError,
  });
  const menuProps = Menu.useContextMenu();
  const { intoEvent } = useCopy();
  // Chromium keeps a text selection elsewhere after a click on a row, and sends Cmd+C
  // to that selection. Dropping it sends Cmd+C to the tree.
  const handleFocus = (e: FocusEvent<HTMLDivElement>) => {
    const text = window.getSelection();
    if (text == null || text.isCollapsed) return;
    if (!text.getRangeAt(0).intersectsNode(e.currentTarget)) text.removeAllRanges();
  };
  const handleCopy = (e: ClipboardEvent<HTMLDivElement>) => {
    // Text selected in the tree, such as part of a name being renamed, copies as text.
    const text = window.getSelection();
    if (
      text != null &&
      !text.isCollapsed &&
      text.getRangeAt(0).intersectsNode(e.currentTarget)
    )
      return;
    const resources = getItem(selectedRef.current);
    if (resources.length === 0) return;
    e.preventDefault();
    intoEvent(resources, e.clipboardData);
  };

  return (
    <Context value={{ store, ...dragAndDrop }}>
      <Menu.ContextMenu menu={renderMenu} {...menuProps} />
      <Base.Tree<string, Resource>
        {...treeProps}
        showRules
        shape={shape}
        subscribe={store.subscribeKey}
        // Not getResource: it throws, and a resource may not be cached before
        // the tree attempts to render it.
        getItem={getItem}
        // The root's children are unknown until its lists answer, and an empty answer
        // alone shows the empty content.
        emptyContent={store.getChildren(rootKey) == null ? null : emptyContent}
        className={className}
        // A draft's row stays mounted, so its rename can start out of view. Focusing
        // the editable then scrolls the row into view.
        pinned={shape.keys.filter(store.isDraft)}
        onContextMenu={menuProps.open}
        onFocus={handleFocus}
        onCopy={handleCopy}
        ref={ref}
        onKeyDown={onKeyDown}
      >
        {itemRenderProp}
      </Base.Tree>
    </Context>
  );
};

interface NoMatchesProps {
  noun: string;
  term: string;
}

const NoMatches = ({ noun, term }: NoMatchesProps): ReactElement => (
  <Text.Text level="small" color={9} className={CSS.BE("tree-search", "message")}>
    No {noun} match “{term}”
  </Text.Text>
);

interface ResultsProps extends Pick<
  Search,
  "noun" | "layout" | "fieldRef" | "resultsRef"
> {
  client: Client;
  root: resource.ID;
  answer: SearchAnswer;
}

const Results = ({
  client,
  root,
  noun,
  layout,
  answer,
  fieldRef,
  resultsRef,
}: ResultsProps): ReactElement => {
  // A second Escape, in the field, closes the search.
  const handleKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    if (e.key !== "Escape" || e.target !== e.currentTarget) return;
    e.stopPropagation();
    fieldRef.current?.focus();
  };
  if (answer.variant === "error")
    return (
      <Status.Summary
        className={CSS.BE("tree-search", "message")}
        variant="error"
        message={`Failed to search ${noun}`}
        description={answer.description}
      />
    );
  return (
    <MatchProvider value={answer.match}>
      <Internal
        key={layout}
        client={client}
        root={root}
        results={answer.results}
        ref={resultsRef}
        onKeyDown={handleKeyDown}
        emptyContent={<NoMatches noun={noun} term={answer.match.term} />}
      />
      {answer.more && (
        <Text.Text level="small" color={9} className={CSS.BE("tree-search", "more")}>
          Showing the first {LIMIT} matches. Type more to narrow the search.
        </Text.Text>
      )}
    </MatchProvider>
  );
};

export interface TreeProps {
  root?: resource.ID | null;
  /**
   * Lists the children of the root. Defaults to the children the item of the root's
   * type lists, for a root that is a resource.
   */
  listRoot?: (client: Client) => Source[];
  emptyContent?: ReactNode;
  /** The menu for a right-click on empty space. */
  rootContextMenu?: FC<DefaultContextMenuProps>;
  /** Shows the search answer in place of the tree while the search has one. */
  search?: Search;
}

export const Tree = ({
  root,
  listRoot,
  emptyContent,
  rootContextMenu,
  search,
}: TreeProps): ReactElement | null => {
  const client = Synnax.use();
  if (root == null || client == null) return null;
  const rootKey = resource.idToString(root);
  return (
    <>
      {/* The results come ahead of the hidden tree, so a lookup by item ID, such as
          the start of a rename, finds the result first. */}
      {search?.answer != null && (
        <Results
          client={client}
          root={root}
          noun={search.noun}
          layout={search.layout}
          answer={search.answer}
          fieldRef={search.fieldRef}
          resultsRef={search.resultsRef}
        />
      )}
      <Internal
        key={rootKey}
        client={client}
        root={root}
        listRoot={listRoot}
        ref={search?.treeRef}
        emptyContent={emptyContent}
        rootContextMenu={rootContextMenu}
        className={search?.answer != null ? CSS.BM("tree", "hidden") : undefined}
      />
    </>
  );
};
