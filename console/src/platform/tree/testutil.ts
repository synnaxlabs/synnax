// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, type Synnax } from "@synnaxlabs/client";
import { type Status, Tree as PTree } from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";

import { type Tree } from "@/platform/tree";
import { type Session } from "@/session";

/**
 * Builds a {@link Tree.TreeState} backed by the given resources. getResource resolves
 * against the supplied resources; the mutating members are no-ops unless overridden.
 */
export const createState = (
  resources: ontology.Resource[],
  overrides: Partial<Tree.TreeState> = {},
): Tree.TreeState => {
  const byKey = new Map(resources.map((r) => [ontology.idToString(r.id), r]));
  const resolve = (id: ontology.ID | string): ontology.Resource => {
    const key = typeof id === "string" ? id : ontology.idToString(id);
    const resource = byKey.get(key);
    if (resource == null) throw new Error(`resource ${key} not found in test state`);
    return resource;
  };
  const getResource = ((id: ontology.ID | string | (ontology.ID | string)[]) =>
    Array.isArray(id) ? id.map(resolve) : resolve(id)) as Tree.GetResource;
  const nodes = resources.map((r) => ({ key: ontology.idToString(r.id) }));
  return {
    nodes,
    shape: PTree.flatten({ nodes, expanded: [] }),
    setNodes: () => {},
    setResource: () => {},
    getResource,
    setSelection: () => {},
    expand: () => {},
    contract: () => {},
    setLoading: () => {},
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
 * context message.
 */
export const createExecutingHandleError =
  (onError: (message: string, exc: unknown) => void = () => {}): Status.ErrorHandler =>
  (excOrFunc, message = "error") => {
    if (typeof excOrFunc !== "function") return onError(message, excOrFunc);
    void (async () => {
      try {
        await excOrFunc();
      } catch (exc) {
        onError(message, exc);
      }
    })();
  };

export interface CreateSelectionArgs {
  ids: ontology.ID[];
  rootID?: ontology.ID;
  parentID?: ontology.ID;
}

/** Builds the selection portion of {@link Tree.ContextMenuProps}. */
export const createSelection = ({
  ids,
  rootID = ontology.ROOT_ID,
  parentID = rootID,
}: CreateSelectionArgs): Tree.ContextMenuProps["selection"] => ({
  ids: array.toArray(ids),
  rootID,
  parentID,
});

export interface CreateBasePropsArgs {
  client: Synnax;
  store: Session.Store;
  overrides?: Partial<Tree.BaseProps>;
}

/**
 * Builds {@link Tree.BaseProps} with the given client and store. The layout and
 * status callbacks default to no-ops; pass overrides (e.g. injected vi.fn() spies) to
 * observe them.
 */
export const createBaseProps = ({
  client,
  store,
  overrides = {},
}: CreateBasePropsArgs): Tree.BaseProps => ({
  client,
  store,
  placeLayout: () => ({ windowKey: "", key: "" }),
  removeLayout: () => undefined,
  addStatus: () => undefined,
  handleError: () => undefined,
  ...overrides,
});
