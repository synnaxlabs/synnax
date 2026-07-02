// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, type Synnax } from "@synnaxlabs/client";
import { Tree } from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";

import { Ontology } from "@/platform/ontology";
import { type Session } from "@/session";

/**
 * Builds a {@link Ontology.Services} registry that returns a NOOP service (carrying the
 * accessed resource type) for every resource type, so the tree and context menus can
 * render against any resource. Pass overrides to supply real service behavior for
 * specific types.
 */
export const buildServices = (
  overrides: Partial<Record<ontology.ResourceType, Ontology.Service>> = {},
): Ontology.Services =>
  new Proxy({} as Ontology.Services, {
    get: (_target, prop) => {
      if (typeof prop !== "string") return undefined;
      const type = prop as ontology.ResourceType;
      return overrides[type] ?? { ...Ontology.NOOP_SERVICE, type };
    },
  });

/**
 * Builds a {@link Ontology.TreeState} backed by the given resources. getResource resolves
 * against the supplied resources; the mutating members are no-ops unless overridden.
 */
export const buildState = (
  resources: ontology.Resource[],
  overrides: Partial<Ontology.TreeState> = {},
): Ontology.TreeState => {
  const byKey = new Map(resources.map((r) => [ontology.idToString(r.id), r]));
  const resolve = (id: ontology.ID | string): ontology.Resource => {
    const key = typeof id === "string" ? id : ontology.idToString(id);
    const resource = byKey.get(key);
    if (resource == null) throw new Error(`resource ${key} not found in test state`);
    return resource;
  };
  const getResource = ((id: ontology.ID | string | (ontology.ID | string)[]) =>
    Array.isArray(id) ? id.map(resolve) : resolve(id)) as Ontology.GetResource;
  const nodes = resources.map((r) => ({ key: ontology.idToString(r.id) }));
  return {
    nodes,
    shape: Tree.flatten({ nodes, expanded: [] }),
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
export const buildResource = (
  id: ontology.ID,
  name: string,
  data?: Record<string, unknown>,
): ontology.Resource => ({
  key: ontology.idToString(id),
  id,
  name,
  ...(data != null ? { data } : {}),
});

export interface BuildSelectionArgs {
  ids: ontology.ID[];
  rootID?: ontology.ID;
  parentID?: ontology.ID;
}

/** Builds the selection portion of {@link Ontology.TreeContextMenuProps}. */
export const buildSelection = ({
  ids,
  rootID = ontology.ROOT_ID,
  parentID = rootID,
}: BuildSelectionArgs): Ontology.TreeContextMenuProps["selection"] => ({
  ids: array.toArray(ids),
  rootID,
  parentID,
});

export interface BuildBasePropsArgs {
  client: Synnax;
  store: Session.Store;
  services?: Ontology.Services;
  overrides?: Partial<Ontology.BaseProps>;
}

/**
 * Builds {@link Ontology.BaseProps} with the given client and store. The layout and
 * status callbacks default to no-ops; pass overrides (e.g. injected vi.fn() spies) to
 * observe them.
 */
export const buildBaseProps = ({
  client,
  store,
  services = buildServices(),
  overrides = {},
}: BuildBasePropsArgs): Ontology.BaseProps => ({
  client,
  store,
  services,
  placeLayout: () => ({ windowKey: "", key: "" }),
  removeLayout: () => undefined,
  addStatus: () => undefined,
  handleError: () => undefined,
  ...overrides,
});
