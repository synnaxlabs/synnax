// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, type Synnax } from "@synnaxlabs/client";
import { Haul } from "@synnaxlabs/lyra/haul";
import { type Status } from "@synnaxlabs/lyra/status";
import { Tree as PTree } from "@synnaxlabs/lyra/tree";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { array } from "@synnaxlabs/x";
import {
  act,
  fireEvent,
  render,
  type RenderOptions,
  type RenderResult,
  screen,
  within,
} from "@testing-library/react";
import { type PropsWithChildren, type ReactElement, type ReactNode } from "react";

import { Errors } from "@/platform/errors";
import { Modals } from "@/platform/modals";
import { Tree } from "@/platform/tree";
import { type Session } from "@/session";
import {
  createConsoleWrapper,
  createTestStore,
  renderSuspended,
  selectTestProject,
  type TestStore,
} from "@/testutil";

/**
 * Builds a {@link Tree.TreeState} backed by the given resources. getResource resolves
 * against the supplied resources; the mutating members are no-ops unless overridden.
 */
export const createState = (
  resources: ontology.Resource[],
  overrides: Partial<Tree.TreeState> = {},
): Tree.TreeState => {
  const byKey = new Map(resources.map((r) => [ontology.idToString(r.id), r]));
  const lookup = (id: ontology.ID | string): ontology.Resource | undefined =>
    byKey.get(typeof id === "string" ? id : ontology.idToString(id));
  const resolve = (id: ontology.ID | string): ontology.Resource => {
    const resource = lookup(id);
    if (resource != null) return resource;
    const key = typeof id === "string" ? id : ontology.idToString(id);
    throw new Error(`resource ${key} not found in test state`);
  };
  // Like the tree's, the list form leaves out the IDs it holds no resource for.
  const getResource = ((id: ontology.ID | string | (ontology.ID | string)[]) =>
    Array.isArray(id)
      ? id.map(lookup).filter((r) => r != null)
      : resolve(id)) as Tree.GetResource;
  const nodes = resources.map((r) => ({ key: ontology.idToString(r.id) }));
  return {
    nodes,
    shape: PTree.flatten({ nodes, expanded: [] }),
    setNodes: () => {},
    setResource: () => {},
    getResource,
    subscribeResource: () => () => {},
    setSelection: () => {},
    expand: () => {},
    contract: () => {},
    ...overrides,
  };
};

/** Builds an ontology resource for the given id with an optional name and data payload. */
export const createResource = (
  id: ontology.ID,
  name: string,
  data?: Record<string, unknown>,
): ontology.Resource => ({
  key: ontology.idToString(id),
  id,
  name,
  ...(data != null ? { data } : {}),
});

/**
 * Builds a {@link Status.ErrorHandler} that actually runs the wrapped operation, so
 * handlers like Tree.HandleSelect (which route their async work through
 * handleError) execute in tests. Failures are reported to onError with the handler's
 * heading.
 */
export const createExecutingHandleError =
  (
    onError: (heading: Status.Heading, exc: unknown) => void = () => {},
  ): Status.ErrorHandler =>
  (excOrFunc, heading = "error") => {
    if (typeof excOrFunc !== "function") return onError(heading, excOrFunc);
    void (async () => {
      try {
        await excOrFunc();
      } catch (exc) {
        onError(heading, exc);
      }
    })();
  };

export interface CreateSelectionParams {
  ids: ontology.ID[];
  rootID?: ontology.ID;
  parentID?: ontology.ID;
}

/** Builds the selection portion of {@link Tree.ContextMenuProps}. */
export const createSelection = ({
  ids,
  rootID = ontology.ROOT_ID,
  parentID = rootID,
}: CreateSelectionParams): Tree.ContextMenuProps["selection"] => ({
  ids: array.toArray(ids),
  rootID,
  parentID,
});

export interface CreateBasePropsParams {
  client: Synnax;
  store: Session.Store;
  overrides?: Partial<Tree.BaseProps>;
}

/**
 * Builds {@link Tree.BaseProps} with the given client and store. The tab and
 * status callbacks default to no-ops; pass overrides (e.g. injected vi.fn() spies) to
 * observe them.
 */
export const createBaseProps = ({
  client,
  store,
  overrides = {},
}: CreateBasePropsParams): Tree.BaseProps => ({
  client,
  store,
  openTab: () => undefined,
  addStatus: () => undefined,
  handleError: () => undefined,
  ...overrides,
});

export interface RenderOntologyTreeOptions extends Pick<
  RenderOptions,
  "onCaughtError"
> {
  client: Synnax;
  root: ontology.ID | null;
  /** Real item overrides; every other resource type falls back to Tree.DefaultItem. */
  items?: Tree.Items;
  /** Extra siblings rendered alongside the tree, e.g. CaptureStatuses. */
  extra?: ReactNode;
  /** The client that creates the session's project; defaults to the rendering one. */
  projectClient?: Synnax;
}

export interface OntologyTreeHandle extends RenderResult {
  store: TestStore;
}

/**
 * Renders the real Tree.Tree against the live Core inside the full console
 * provider stack, with a mounted modal stack so context-menu flows can open prompts.
 * The Triggers provider is included so held-modifier interactions (control-click
 * multi-select) behave as they do in the app. An active project is established so
 * onSelect handlers that place a tab have a project to create panels under.
 */
export const renderOntologyTree = async ({
  client,
  root,
  items = {},
  extra,
  projectClient = client,
  onCaughtError,
}: RenderOntologyTreeOptions): Promise<OntologyTreeHandle> => {
  const { wrapper, store } = await createConsoleWrapper({ client });
  await selectTestProject(store, projectClient);
  const rendered = render(
    <Triggers.Provider>
      <Haul.Provider>
        <Tree.Provider items={items}>
          <Tree.Tree root={root} />
          <Modals.Stack />
        </Tree.Provider>
      </Haul.Provider>
      {extra}
    </Triggers.Provider>,
    { wrapper, onCaughtError },
  );
  return { ...rendered, store };
};

const SCROLL_STEP = 200;

/**
 * Finds the tree label matching text, scrolling the tree when the row sits outside the
 * window. The tree keeps only the rows near the window mounted, so a label further down
 * reaches the DOM the same way a user reaches it. Leaves the tree scrolled where the
 * label was found, and back at the top when there is none.
 */
// Sweeps from the top: a previous lookup can leave the tree scrolled past the row.
const sweepTree = async (text: string): Promise<HTMLElement | null> => {
  const tree = document.querySelector<HTMLElement>(".lyra-tree");
  const rows = tree?.querySelector<HTMLElement>(".lyra-list__virtualizer");
  if (tree == null || rows == null) return null;
  const total = parseInt(rows.style.minHeight, 10);
  for (let offset = 0; offset <= total; offset += SCROLL_STEP) {
    await act(async () => {
      tree.scrollTop = offset;
      fireEvent.scroll(tree);
    });
    const label = screen.queryByText(text);
    if (label != null) return label;
  }
  return null;
};

const findTreeLabel = async (text: string): Promise<HTMLElement> => {
  const mounted = screen.queryByText(text);
  if (mounted != null) return mounted;
  const swept = await sweepTree(text);
  if (swept != null) return swept;
  // Absent from the data rather than the window: wait for it to load, then sweep again.
  try {
    return await screen.findByText(text);
  } catch (err) {
    const retried = await sweepTree(text);
    if (retried != null) return retried;
    throw err instanceof Error ? err : new Error(String(err), { cause: err });
  }
};

/** Finds the rendered tree row containing the given text. */
export const findTreeRow = async (text: string): Promise<Element> => {
  const row = (await findTreeLabel(text)).closest(".lyra-tree__item");
  if (row == null) throw new Error(`tree row for ${text} not found`);
  return row;
};

/** Right-clicks the tree row containing text, opening its service context menu. A
 * menu that suspends must mount inside an async act: a suspension mounted under
 * sync act never retries when its promise resolves, so the menu stays empty. */
export const openTreeRowContextMenu = async (text: string): Promise<void> => {
  const row = await findTreeRow(text);
  await act(async () => {
    fireEvent.contextMenu(row);
  });
};

export interface RenderTreeContextMenuOptions extends Pick<
  RenderOptions,
  "onCaughtError"
> {
  client: Synnax;
  /** Resources backing the tree state. */
  resources: ontology.Resource[];
  /** The selected ids; defaults to every resource. */
  selected?: ontology.ID[];
  /** The tree parent of the selection; defaults to the root. */
  parentID?: ontology.ID;
  store?: TestStore;
  baseOverrides?: Partial<Tree.BaseProps>;
  stateOverrides?: Partial<Tree.TreeState>;
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
    parentID,
    store,
    baseOverrides = {},
    stateOverrides = {},
    onCaughtError,
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
      parentID,
    }),
    state: createState(resources, stateOverrides),
  };
  const { wrapper } = await createConsoleWrapper({ client, store: resolvedStore });
  await renderSuspended(
    <>
      <Errors.SuspenseBoundary loading={null}>
        <TreeContextMenu {...props} />
      </Errors.SuspenseBoundary>
      <Modals.Stack />
    </>,
    { wrapper, onCaughtError },
  );
  return { store: resolvedStore, props };
};

export interface RenderToolbarOptions extends Pick<RenderOptions, "onCaughtError"> {
  client: Synnax;
  /** Real tree items keyed by resource type; other types fall back to Tree.DefaultItem. */
  items?: Tree.Items;
}

export interface ToolbarHandle {
  store: TestStore;
}

/**
 * Renders a domain's navdrawer toolbar content inside the full console provider stack
 * with haul + ontology services and a live modal stack, so the toolbar's resource tree
 * and its actions run against the real Core.
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
