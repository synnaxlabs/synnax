// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { patch } from "@synnaxlabs/x";
import { type Draft } from "immer";

import { actions } from "@/actions";
import {
  type Action,
  addEdge,
  createReduceAll,
  type HandlerResult,
  type Handlers,
  removeEdge,
  removeNode,
  set,
  setNode,
  setNodePosition,
} from "@/schematic/actions.gen";
import { type ElementConfig, type ElementField, fieldOf } from "@/schematic/element";
import {
  edgeConfigZ,
  nodeConfigZ,
  type Schematic,
  schematicZ,
} from "@/schematic/types.gen";

const ELEMENT_FIELDS: readonly string[] = ["nodes", "edges"] satisfies ElementField[];

const CONFIG_FIELD = "config";

const PARENT_FIELD = "parent";

const hasNode = (state: Draft<Schematic>, key: string): boolean =>
  state.nodes.some((n) => n.key === key);

const hasEdge = (state: Draft<Schematic>, key: string): boolean =>
  state.edges.some((e) => e.key === key);

// A path into a node or edge goes into its config, or names the parent of a node. An
// action on a node or edge that does not exist does nothing.
const applyPath = (state: Draft<Schematic>, action: patch.Action): HandlerResult => {
  const path = patch.pathOf(action);
  if (!ELEMENT_FIELDS.includes(path[0]))
    return actions.applyPath(state, action, schematicZ);
  if (path[2] !== CONFIG_FIELD && (path[0] !== "nodes" || path[2] !== PARENT_FIELD))
    throw new Error(
      `[Schematic] - a path into ${path[0]} must go into the config of one entry or ` +
        "name the parent of a node",
    );
  return actions.applyEntryPath(state, action, [path[0]], schematicZ);
};

const isConfigEdit = (action: Action): boolean => {
  switch (action.type) {
    case "set":
    case "remove":
      return ELEMENT_FIELDS.includes(patch.pathOf(action)[0]);
    default:
      return false;
  }
};

const handlers: Handlers = {
  create: (state, payload) => {
    Object.assign(state, payload.schematic);
    return { inverse: [], targets: [payload.schematic.key] };
  },

  setNodePosition: (state, payload) => {
    const node = state.nodes.find((n) => n.key === payload.key);
    if (node == null) return actions.NO_OP_RESULT;
    const oldPosition = { x: node.position.x, y: node.position.y };
    node.position = payload.position;
    return {
      inverse: [setNodePosition({ key: payload.key, position: oldPosition })],
      targets: [payload.key],
    };
  },
  setNode: (state, payload) => {
    const idx = state.nodes.findIndex((n) => n.key === payload.node.key);
    if (idx === -1) {
      state.nodes.push(payload.node);
      return {
        inverse: [removeNode({ key: payload.node.key })],
        targets: [payload.node.key],
      };
    }
    const oldNode = actions.snapshotDraft(state.nodes[idx]);
    state.nodes[idx] = payload.node;
    return { inverse: [setNode({ node: oldNode })], targets: [payload.node.key] };
  },
  removeNode: (state, payload) => {
    const idx = state.nodes.findIndex((n) => n.key === payload.key);
    if (idx === -1) return actions.NO_OP_RESULT;
    const oldNode = actions.snapshotDraft(state.nodes[idx]);
    state.nodes.splice(idx, 1);
    const inverse: Action[] = [setNode({ node: oldNode })];
    const targets = [payload.key];
    const connected = state.edges.filter(
      (e) => e.source.node === payload.key || e.target.node === payload.key,
    );
    for (const e of connected) {
      inverse.push(addEdge({ edge: actions.snapshotDraft(e) }));
      targets.push(e.key);
    }
    state.edges = state.edges.filter(
      (e) => e.source.node !== payload.key && e.target.node !== payload.key,
    );
    const { parent } = oldNode;
    for (const n of state.nodes) {
      if (n.parent !== payload.key) continue;
      inverse.push(set({ path: ["nodes", n.key, PARENT_FIELD], value: payload.key }));
      if (parent == null) delete n.parent;
      else n.parent = parent;
      targets.push(n.key);
    }
    return { inverse, targets };
  },
  addEdge: (state, payload) => {
    const { key, source, target } = payload.edge;
    if (
      hasEdge(state, key) ||
      !hasNode(state, source.node) ||
      !hasNode(state, target.node)
    )
      return actions.NO_OP_RESULT;
    state.edges.push(payload.edge);
    return {
      inverse: [removeEdge({ key: payload.edge.key })],
      targets: [payload.edge.key],
    };
  },

  removeEdge: (state, payload) => {
    const idx = state.edges.findIndex((e) => e.key === payload.key);
    if (idx === -1) return actions.NO_OP_RESULT;
    const oldEdge = actions.snapshotDraft(state.edges[idx]);
    state.edges.splice(idx, 1);
    return { inverse: [addEdge({ edge: oldEdge })], targets: [payload.key] };
  },
  ...actions.createPathHandlers(applyPath),
};

const reduce = createReduceAll(handlers);

// The handlers put the objects of an action into the state, where a later action in
// the batch edits them in place. A copy keeps the caller's actions unchanged.
export const reduceAll = (state: Schematic, actions: Action[]) =>
  reduce(state, structuredClone(actions));

export interface SetChangedArgs {
  /** The key of the node or edge. */
  key: string;
  base: ElementConfig;
  next: ElementConfig;
}

/**
 * Writes only the fields of the config of a node or edge that differ between base and
 * next, so a concurrent edit to another field of the element survives.
 * @returns the actions to dispatch, with none when nothing differs.
 */
export const setChanged = ({ key, base, next }: SetChangedArgs): Action[] => {
  const field = fieldOf(base);
  const schema = field === "edges" ? edgeConfigZ : nodeConfigZ;
  return patch
    .diff(schema, base, next)
    .map((a) => patch.prefix(a, [field, key, CONFIG_FIELD]));
};

/**
 * Converts a path action from the camelCase form of the cached schematic into the
 * snake_case form that the Core stores. Other actions keep their form.
 */
export const encode = (state: Schematic, action: Action): Action => {
  switch (action.type) {
    case "set":
    case "remove":
    case "insert":
    case "move":
      return patch.camelToSnake(action, { schema: schematicZ, data: state });
    default:
      return action;
  }
};

// createOf hands the dispatch controller the document carried by a create
// action so frames for never-cached documents ingest instead of drop.
export const createOf = (action: Action) =>
  action.type === "create" ? action.create.schematic : undefined;

export const kindOf = (actions: Action[]): string => {
  if (actions.length === 0) return "default";
  // A drag dispatches a stream of `set_node_position` per frame, plus config edits
  // synthesized by augmentWithEdgeSegments for any affected edges. Both shapes are part
  // of one user gesture and must coalesce together; classify them all as "move" so the
  // per-kind coalesce window collapses them into a single undoable.
  const hasMove = actions.some((a) => a.type === "set_node_position");
  const onlyMoveOrSegment = actions.every(
    (a) => a.type === "set_node_position" || isConfigEdit(a),
  );
  if (hasMove && onlyMoveOrSegment) return "move";
  // One gesture changes a varying number of fields, such as a resize that changes the
  // width on one frame and both sides on the next.
  if (actions.every(isConfigEdit)) return "config";
  if (actions.length === 1) return actions[0].type;
  return "transaction";
};
