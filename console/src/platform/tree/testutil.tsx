// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, group, resource, type Synnax } from "@synnaxlabs/client";
import { Haul } from "@synnaxlabs/lyra/haul";
import { Menu } from "@synnaxlabs/lyra/menu";
import { type Status } from "@synnaxlabs/lyra/status";
import { Tree as PTree } from "@synnaxlabs/lyra/tree";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { array, sleep, TimeSpan } from "@synnaxlabs/x";
import {
  act,
  fireEvent,
  getConfig,
  type Matcher,
  type MatcherFunction,
  render,
  type RenderOptions,
  type RenderResult,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { type PropsWithChildren, type ReactElement, type ReactNode } from "react";

import { Errors } from "@/platform/errors";
import { Modals } from "@/platform/modals";
import { Tree } from "@/platform/tree";
import { type Session } from "@/session";
import {
  CaptureStatuses,
  type ConsolePreloadedState,
  createConsoleWrapper,
  createTestStore,
  getIconButton,
  renderSuspended,
  renderWithConsole,
  selectTestProject,
  type TestStore,
  uniqueName,
  uniqueToken,
} from "@/testutil";

export interface StateOptions extends Partial<Tree.TreeState> {
  /** The nodes the state shows. Defaults to the resources, all at the top level. */
  nodes?: PTree.Node<string>[];
  expanded?: string[];
}

/**
 * Builds a {@link Tree.TreeState} backed by the given resources and nodes. getResource
 * resolves against the resources; the mutating members are no-ops unless overridden.
 */
export const createState = (
  resources: Tree.Resource[],
  {
    nodes = resources.map((r) => ({ key: resource.idToString(r.id) })),
    expanded = [],
    ...overrides
  }: StateOptions = {},
): Tree.TreeState => {
  const byKey = new Map(resources.map((r) => [resource.idToString(r.id), r]));
  const lookup = (id: resource.ID | string): Tree.Resource | undefined =>
    byKey.get(typeof id === "string" ? id : resource.idToString(id));
  const resolve = (id: resource.ID | string): Tree.Resource => {
    const found = lookup(id);
    if (found != null) return found;
    const key = typeof id === "string" ? id : resource.idToString(id);
    throw new Error(`resource ${key} not found in test state`);
  };
  // Like the tree's, the list form leaves out the IDs it holds no resource for.
  const getResource = ((id: resource.ID | string | (resource.ID | string)[]) =>
    Array.isArray(id)
      ? id.map(lookup).filter((r) => r != null)
      : resolve(id)) as Tree.GetResource;
  const children = new Map<string, string[]>();
  const parents = new Map<string, string>();
  const index = (parent: string, level: PTree.Node<string>[]) =>
    level.forEach(({ key, children: below }) => {
      parents.set(key, parent);
      if (below == null) return;
      children.set(
        key,
        below.map((child) => child.key),
      );
      index(key, below);
    });
  index("", nodes);
  return {
    shape: PTree.flatten({ nodes, expanded }),
    getChildren: (key) => children.get(key),
    getParent: (key) => parents.get(key),
    addDraft: () => () => {},
    getResource,
    subscribeResource: () => () => {},
    setSelection: () => {},
    expand: () => {},
    contract: () => {},
    ...overrides,
  };
};

/** Builds a tree resource for the given id and name. */
export const createResource = (id: resource.ID, name: string): Tree.Resource => ({
  key: resource.idToString(id),
  id,
  name,
});

/** Lists the groups whose parent is the node. */
const listChildGroups: Tree.Children = ({ client, id }) => [
  Tree.createSource(client.groups, { parents: [id] }, Tree.toResource("group")),
];

/**
 * Builds a group item that lists its child groups and moves dropped groups into it.
 * Params override any part of the item.
 */
export const createGroupItem = (
  params: Partial<Tree.CreateItemParams> = {},
): Tree.Item =>
  Tree.createItem({
    children: listChildGroups,
    folder: true,
    drop: async ({ client, destination, items }) =>
      await client.groups.setParent(
        items.map(({ id }) => id.key),
        destination.id,
      ),
    ...params,
  });

const listMembers = (client: Synnax, key: group.Key): Tree.Source[] => [
  Tree.createSource(
    client.groups,
    { parents: [group.resourceID(key)] },
    Tree.toResource("group"),
  ),
  Tree.createSource(client.channels, { where: { groups: [key] } }, (ch) =>
    Tree.toResource("channel")(ch.payload),
  ),
];

/** The items of a tree and the lister of its root. */
export interface ScopeTree extends Required<
  Pick<RenderLiveTreeOptions, "items" | "listRoot">
> {}

/**
 * Builds a tree rooted at the channel scope, which offers group creation. The root
 * lists the groups and channels in the container group instead of the whole scope, and
 * so does each group. groupParams override any part of the group item.
 */
export const createScopeTree = (
  container: group.Key,
  groupParams: Partial<Tree.CreateItemParams> = {},
): ScopeTree => ({
  items: {
    group: createGroupItem({
      children: ({ client, id }) => listMembers(client, id.key),
      ...groupParams,
    }),
  },
  listRoot: (client) => listMembers(client, container),
});

/** The groups that {@link createSearchTree} makes. */
export interface SearchTree {
  root: resource.ID;
  bravo: group.Key;
  tok: string;
  names: Record<
    "alpha" | "bravo" | "charlie" | "delta" | "echo" | "golf" | "hotel",
    string
  >;
}

// root > alpha > {bravo > {charlie, hotel}, golf}, root > delta, root > echo. Only
// alpha, charlie, and delta match the token.
export const createSearchTree = async (client: Synnax): Promise<SearchTree> => {
  const tok = uniqueToken();
  const names = {
    alpha: `${tok} alpha`,
    bravo: uniqueName("bravo"),
    charlie: `${tok} charlie`,
    delta: `${tok} delta`,
    echo: uniqueName("echo"),
    golf: uniqueName("golf"),
    hotel: uniqueName("hotel"),
  };
  const root = await client.groups.create({
    scope: "channel",
    name: uniqueName("root"),
  });
  const rootID = group.resourceID(root.key);
  const alpha = await client.groups.create({ parent: rootID, name: names.alpha });
  const alphaID = group.resourceID(alpha.key);
  const bravo = await client.groups.create({ parent: alphaID, name: names.bravo });
  const bravoID = group.resourceID(bravo.key);
  await client.groups.create({ parent: bravoID, name: names.charlie });
  await client.groups.create({ parent: bravoID, name: names.hotel });
  await client.groups.create({ parent: alphaID, name: names.golf });
  await client.groups.create({ parent: rootID, name: names.delta });
  await client.groups.create({ parent: rootID, name: names.echo });
  return { root: rootID, bravo: bravo.key, tok, names };
};

/** The root of a tree whose children come from {@link createHostItems}. */
export const HOST_ID: resource.ID = { type: "builtin", key: "host" };

/**
 * Builds the item of a tree rooted at HOST_ID that lists the records of the sources,
 * for specs whose records sit in no group.
 */
export const createHostItems = (
  children: (client: Synnax) => Tree.Source[],
): Tree.Items => ({
  [HOST_ID.type]: Tree.createItem({
    children: ({ client }) => children(client),
  }),
});

/** Returns the keys of the groups whose parent is the given resource. */
export const retrieveChildGroupKeys = async (
  client: Synnax,
  parent: resource.ID,
): Promise<string[]> =>
  (await client.groups.retrieve({ parents: [parent] })).map(({ key }) => key);

export interface CreateSelectionParams {
  ids: resource.ID[];
  rootID?: resource.ID;
  parentID?: resource.ID;
}

/** Builds the selection portion of {@link Tree.ContextMenuProps}. */
export const createSelection = ({
  ids,
  rootID = channel.TYPE_RESOURCE_ID,
  parentID = rootID,
}: CreateSelectionParams): Tree.ContextMenuProps["selection"] => ({
  ids: array.toArray(ids),
  top: array.toArray(ids),
  rootID,
  parentID,
});

export interface CreateBasePropsParams {
  client: Synnax;
  store: Session.Store;
  overrides?: Partial<Tree.BaseProps>;
}

/**
 * Builds {@link Tree.BaseProps} with the given client and store. The tab and error
 * callbacks default to no-ops. Pass overrides, such as vi.fn() spies, to observe them.
 */
export const createBaseProps = ({
  client,
  store,
  overrides = {},
}: CreateBasePropsParams): Tree.BaseProps => ({
  client,
  store,
  openTab: () => undefined,
  handleError: () => undefined,
  ...overrides,
});

export interface RenderCopyContextMenuItemOptions {
  /** Resources backing the tree state. */
  resources: Tree.Resource[];
  /** The selected ids; defaults to every resource. */
  selected?: resource.ID[];
  items?: Tree.Items;
  preloadedState?: ConsolePreloadedState;
}

/**
 * Renders Tree.CopyContextMenuItem in a menu. Returns a getter for the statuses it
 * raises.
 */
export const renderCopyContextMenuItem = async ({
  resources,
  selected,
  items = {},
  preloadedState,
}: RenderCopyContextMenuItemOptions): Promise<() => Status.NotificationSpec[]> => {
  const store = await createTestStore({ preloadedState });
  const props: Tree.CopyContextMenuItemProps = {
    selection: createSelection({ ids: selected ?? resources.map(({ id }) => id) }),
    state: createState(resources),
  };
  let statuses: Status.NotificationSpec[] = [];
  await renderWithConsole(
    <Tree.Provider items={items}>
      <Menu.Menu>
        <Tree.CopyContextMenuItem {...props} />
      </Menu.Menu>
      <CaptureStatuses onStatuses={(next) => (statuses = next)} />
    </Tree.Provider>,
    { store },
  );
  return () => statuses;
};

export interface RenderLiveTreeOptions extends Pick<RenderOptions, "onCaughtError"> {
  client: Synnax;
  root: resource.ID | null;
  /** Real item overrides; every other resource type falls back to Tree.DefaultItem. */
  items?: Tree.Items;
  listRoot?: Tree.TreeProps["listRoot"];
  /** Extra siblings rendered alongside the tree, e.g. CaptureStatuses. */
  extra?: ReactNode;
  /** The client that creates the session's project; defaults to the rendering one. */
  projectClient?: Synnax;
  /** Renders a search action and field above the tree. */
  search?: Tree.UseSearch;
  preloadedState?: ConsolePreloadedState;
  contextMenus?: Tree.ContextMenus;
}

export interface LiveTreeHandle extends RenderResult {
  store: TestStore;
}

/**
 * Renders the real Tree.Tree against the live Core inside the full console
 * provider stack, with a mounted modal stack so context-menu flows can open prompts.
 * The Triggers provider is included so held-modifier interactions (control-click
 * multi-select) behave as they do in the app. An active project is established so
 * onSelect handlers that place a tab have a project to create panels under.
 */
interface SearchableTreeProps extends Pick<Tree.TreeProps, "listRoot"> {
  root: resource.ID | null;
}

const createSearchableTree = (useSearch: Tree.UseSearch) => {
  const SearchableTree = ({ root, listRoot }: SearchableTreeProps): ReactElement => {
    const search = useSearch(root);
    return (
      <>
        {search.open ? (
          <Tree.SearchField search={search} />
        ) : (
          <Tree.SearchAction search={search} />
        )}
        <Tree.Tree root={root} listRoot={listRoot} search={search} />
      </>
    );
  };
  return SearchableTree;
};

export const renderLiveTree = async ({
  client,
  root,
  items = {},
  listRoot,
  extra,
  projectClient = client,
  search,
  contextMenus,
  onCaughtError,
  preloadedState,
}: RenderLiveTreeOptions): Promise<LiveTreeHandle> => {
  const { wrapper, store } = await createConsoleWrapper({ client, preloadedState });
  await selectTestProject(store, projectClient);
  const SearchableTree = search == null ? null : createSearchableTree(search);
  const rendered = render(
    <Triggers.Provider>
      <Haul.Provider>
        <Tree.Provider items={items} contextMenus={contextMenus}>
          {SearchableTree == null ? (
            <Tree.Tree root={root} listRoot={listRoot} />
          ) : (
            <SearchableTree root={root} listRoot={listRoot} />
          )}
          <Modals.Stack />
        </Tree.Provider>
      </Haul.Provider>
      {extra}
    </Triggers.Provider>,
    { wrapper, onCaughtError },
  );
  return { ...rendered, store };
};

/** Renders a tree of groups under root that searches with search. */
export const renderSearchableTree = async (
  client: Synnax,
  root: resource.ID,
  search: Tree.UseSearch,
): Promise<LiveTreeHandle> =>
  await renderLiveTree({ client, root, items: { group: createGroupItem() }, search });

/** Returns how many trees an open search hides. */
export const countHiddenTrees = (): number =>
  document.querySelectorAll(".lyra-tree.console-tree--hidden").length;

const SCROLL_STEP = 200;
// A search hides the browse tree, so the helpers skip it as a user does.
const VISIBLE_TREE = ".lyra-tree:not(.console-tree--hidden)";
const QUERY_OPTIONS = { ignore: "script, style, .console-tree--hidden *" };

const SWEEP_INTERVAL = TimeSpan.milliseconds(50);

const rowsHeight = (): number => {
  const rows = document.querySelector<HTMLElement>(
    `${VISIBLE_TREE} .lyra-list__virtualizer`,
  );
  return rows == null ? 0 : parseInt(rows.style.minHeight, 10);
};

// Sweeps from the top by default: a previous lookup can leave the tree scrolled past
// the row. Rows can land during the sweep, so it reads the height at each step.
const sweepTree = async (text: Matcher, from = 0): Promise<HTMLElement | null> => {
  const tree = document.querySelector<HTMLElement>(VISIBLE_TREE);
  if (tree == null) return null;
  for (let offset = from; offset <= rowsHeight(); offset += SCROLL_STEP) {
    await act(async () => {
      tree.scrollTop = offset;
      fireEvent.scroll(tree);
    });
    const label = screen.queryByText(text, QUERY_OPTIONS);
    if (label != null) return label;
  }
  return null;
};

const SHOW_MORE = "Show more";
const MAX_PAGES = 20;
const BOTTOM_SPAN = 5 * SCROLL_STEP;

// Loads the next page of a folder, as a user does, until the label shows. A page that
// lands where the show more node was, so the label sweep starts there. The last folder
// pages near the bottom, so the node sweep starts there. A tree with no rows has no
// page to load. Rows that land during its node sweeps would only pass under them and
// use up the timeout, so the next label sweep looks for them instead.
const pageTo = async (text: Matcher): Promise<HTMLElement | null> => {
  if (rowsHeight() === 0) return null;
  for (let page = 0; page < MAX_PAGES; page++) {
    const height = rowsHeight();
    const bottom = Math.max(0, height - BOTTOM_SPAN);
    const more = (await sweepTree(SHOW_MORE, bottom)) ?? (await sweepTree(SHOW_MORE));
    if (more == null) return null;
    const at = document.querySelector<HTMLElement>(VISIBLE_TREE)?.scrollTop ?? 0;
    await act(async () => {
      fireEvent.click(more);
    });
    await waitFor(() => {
      if (more.isConnected && rowsHeight() === height)
        throw new Error("the next page has not landed");
    });
    const found = await sweepTree(text, at);
    if (found != null) return found;
  }
  return null;
};

/**
 * Finds the tree label matching text, scrolling the tree when the row sits outside the
 * window and loading the next pages of a folder. The tree keeps only the rows near the
 * window mounted, so a label further down reaches the DOM the same way a user reaches
 * it. Tries again until the label loads or the async util timeout passes. Leaves the
 * tree scrolled where the label was found.
 */
const findTreeLabel = async (text: Matcher): Promise<HTMLElement> => {
  const deadline = performance.now() + getConfig().asyncUtilTimeout;
  const attempt = async () => (await sweepTree(text)) ?? (await pageTo(text));
  let label = screen.queryByText(text, QUERY_OPTIONS) ?? (await attempt());
  while (label == null && performance.now() < deadline) {
    await act(async () => await sleep.sleep(SWEEP_INTERVAL));
    label = await attempt();
  }
  return label ?? screen.getByText(text, QUERY_OPTIONS);
};

/**
 * Matches the whole name of a tree item, including a search result name that the
 * search tints in parts.
 */
export const itemName =
  (name: string): MatcherFunction =>
  (_, el) =>
    el?.matches(".lyra-text:not(.lyra-tree__item)") === true && el.textContent === name;

/** Finds the rendered tree row containing the given text. */
export const findTreeRow = async (text: Matcher): Promise<Element> => {
  const row = (await findTreeLabel(text)).closest(".lyra-tree__item");
  if (row == null) throw new Error(`tree row for ${String(text)} not found`);
  return row;
};

/** Right-clicks the tree row containing text, opening its service context menu. A
 * menu that suspends must mount inside an async act: a suspension mounted under
 * sync act never retries when its promise resolves, so the menu stays empty. */
export const openTreeRowContextMenu = async (text: Matcher): Promise<void> => {
  const row = await findTreeRow(text);
  await act(async () => {
    fireEvent.contextMenu(row);
  });
};

/** Returns the icon named by its registry name inside the tree row, or null. */
export const findTreeRowIcon = async (
  text: string,
  icon: string,
): Promise<Element | null> =>
  (await findTreeRow(text)).querySelector(`.lyra-icon--${icon}`);

/** Right-clicks the tree's empty space, opening the root context menu. */
export const openTreeRootContextMenu = async (): Promise<void> => {
  const tree = await waitFor(() => {
    const el = document.querySelector(".lyra-tree");
    if (el == null) throw new Error("tree not found");
    return el;
  });
  await act(async () => {
    fireEvent.contextMenu(tree);
  });
};

export interface RenderTreeContextMenuOptions extends Pick<
  RenderOptions,
  "onCaughtError"
> {
  client: Synnax;
  /** Resources backing the tree state. */
  resources: Tree.Resource[];
  /** The selected ids; defaults to every resource. */
  selected?: resource.ID[];
  /** The tree root; defaults to the channel type ID. */
  rootID?: resource.ID;
  /** The tree parent of the selection; defaults to the root. */
  parentID?: resource.ID;
  store?: TestStore;
  baseOverrides?: Partial<Tree.BaseProps>;
  stateOverrides?: StateOptions;
  /** Extra siblings rendered alongside the menu, e.g. CaptureStatuses. */
  extra?: ReactNode;
  /** The tree items the menu reads, as the tree provides them. */
  items?: Tree.Items;
}

export interface TreeContextMenuHandle {
  store: TestStore;
  props: Tree.ContextMenuProps;
}

/**
 * Renders a domain's TreeContextMenu against the full console provider stack with a
 * live modal stack, backed by the given resources and selection. Returns the store and
 * the exact props handed to the menu so specs can re-invoke handlers or inspect state.
 */
export const renderTreeContextMenu = async (
  TreeContextMenu: Tree.ContextMenu,
  {
    client,
    resources,
    selected,
    rootID,
    parentID,
    store,
    baseOverrides = {},
    stateOverrides = {},
    onCaughtError,
    extra,
    items = {},
  }: RenderTreeContextMenuOptions,
): Promise<TreeContextMenuHandle> => {
  const resolvedStore = store ?? (await createTestStore());
  const props: Tree.ContextMenuProps = {
    ...createBaseProps({
      client,
      store: resolvedStore,
      overrides: baseOverrides,
    }),
    selection: createSelection({
      ids: selected ?? resources.map((r) => r.id),
      rootID,
      parentID,
    }),
    state: createState(resources, stateOverrides),
  };
  const { wrapper } = await createConsoleWrapper({ client, store: resolvedStore });
  await renderSuspended(
    <Tree.Provider items={items}>
      <Errors.SuspenseBoundary loading={null}>
        <TreeContextMenu {...props} />
      </Errors.SuspenseBoundary>
      <Modals.Stack />
      {extra}
    </Tree.Provider>,
    { wrapper, onCaughtError },
  );
  return { store: resolvedStore, props };
};

export interface RenderToolbarOptions extends Pick<RenderOptions, "onCaughtError"> {
  client: Synnax;
  /**
   * Real tree items keyed by resource type; other types fall back to Tree.DefaultItem.
   */
  items?: Tree.Items;
}

export interface ToolbarHandle {
  store: TestStore;
}

/**
 * Renders a domain's navdrawer toolbar content inside the full console provider stack
 * with haul and tree items and a live modal stack, so the toolbar's resource tree and
 * its actions run against the real Core.
 */
export const renderToolbar = async (
  content: ReactNode,
  { client, items, onCaughtError }: RenderToolbarOptions,
): Promise<ToolbarHandle> => {
  const { wrapper: Console, store } = await createConsoleWrapper({ client });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <Haul.Provider>
        <Tree.Provider items={items ?? {}}>{children}</Tree.Provider>
      </Haul.Provider>
    </Console>
  );
  Wrapper.displayName = "ToolbarWrapper";
  await renderSuspended(
    <>
      {content}
      <Modals.Stack />
    </>,
    { wrapper: Wrapper, onCaughtError },
  );
  return { store };
};

/** Opens the search of a tree or toolbar and types the term. */
export const searchTree = (term: string): void => {
  fireEvent.click(getIconButton(document.body, "search"));
  fireEvent.change(screen.getByPlaceholderText(/^Search /), {
    target: { value: term },
  });
};

/** Returns the tree row (`.lyra-tree__item`) whose label matches text. */
export const getTreeRow = (text: string): HTMLElement => {
  const row = screen.getByText(text).closest<HTMLElement>(".lyra-tree__item");
  if (row == null) throw new Error(`tree row for ${text} not found`);
  return row;
};

/**
 * Clicks the expansion caret on the tree row matching text, loading its children.
 * Tree groups lazy-load their children only once expanded.
 */
export const expandTreeRow = (text: string): void => {
  const caret = getTreeRow(text).querySelector(".lyra-tree__expansion-indicator");
  if (caret == null) throw new Error(`expansion indicator for ${text} not found`);
  fireEvent.click(caret);
};

/**
 * Finds the button containing text inside the open modal dialog. Context menus render
 * items as buttons too, so an unscoped text query can hit the menu item that opened
 * the modal instead of the modal's own action.
 */
export const findModalButton = (text: string): HTMLButtonElement => {
  const dialog = screen.getByRole("dialog");
  const btn = within(dialog)
    .getAllByText(text)
    .map((el) => el.closest<HTMLButtonElement>("button"))
    .find((b) => b != null);
  if (btn == null) throw new Error(`modal button with text ${text} not found`);
  return btn;
};
