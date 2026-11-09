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
  addEdge,
  createReduceAll,
  type HandlerResult,
  type Handlers,
  removeEdge,
  removeNode,
  setConfig,
  setNode,
  setNodePosition,
} from "@/schematic/actions.gen";
import {
  type Action,
  type ElementConfig,
  elementConfigZ,
  type Schematic,
  schematicZ,
} from "@/schematic/types.gen";

const CONFIGS_FIELD = "configs";

const hasNode = (state: Draft<Schematic>, key: string): boolean =>
  state.nodes.some((n) => n.key === key);

const hasEdge = (state: Draft<Schematic>, key: string): boolean =>
  state.edges.some((e) => e.key === key);

// Only the key of a node or edge can name a config.
const hasElement = (state: Draft<Schematic>, key: string): boolean =>
  hasNode(state, key) || hasEdge(state, key);

// An action on the config of a node or edge that does not exist does nothing.
const applyPath = (state: Draft<Schematic>, action: patch.Action): HandlerResult => {
  const path = patch.pathOf(action);
  if (path[0] !== CONFIGS_FIELD) return actions.applyPath(state, action, schematicZ);
  if (path.length > 1 && !hasElement(state, path[1])) return actions.NO_OP_RESULT;
  return actions.applyEntryPath(state, action, [CONFIGS_FIELD], schematicZ);
};

const snapshotConfig = (
  state: Draft<Schematic>,
  key: string,
): ElementConfig | undefined => {
  const config = state.configs[key];
  return config != null ? actions.snapshotDraft(config) : undefined;
};

const isConfigEdit = (action: Action): boolean => {
  switch (action.type) {
    case "set_config":
      return true;
    case "set":
    case "remove":
      return patch.pathOf(action)[0] === CONFIGS_FIELD;
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
      if (payload.config != null) state.configs[payload.node.key] = payload.config;
      return {
        inverse: [removeNode({ key: payload.node.key })],
        targets: [payload.node.key],
      };
    }
    const oldNode = actions.snapshotDraft(state.nodes[idx]);
    const oldConfigRaw = state.configs[payload.node.key];
    const oldConfig =
      oldConfigRaw != null ? actions.snapshotDraft(oldConfigRaw) : undefined;
    state.nodes[idx] = payload.node;
    if (payload.config != null) state.configs[payload.node.key] = payload.config;
    return {
      inverse: [
        setNode(
          oldConfig != null
            ? { node: oldNode, config: oldConfig }
            : { node: oldNode, config: undefined },
        ),
      ],
      targets: [payload.node.key],
    };
  },
  removeNode: (state, payload) => {
    const idx = state.nodes.findIndex((n) => n.key === payload.key);
    if (idx === -1) return actions.NO_OP_RESULT;
    const oldNode = actions.snapshotDraft(state.nodes[idx]);
    const oldConfig = snapshotConfig(state, payload.key);
    state.nodes.splice(idx, 1);
    delete state.configs[payload.key];
    const inverse: Action[] = [setNode({ node: oldNode, config: oldConfig })];
    const targets = [payload.key];
    const connected = state.edges.filter(
      (e) => e.source.node === payload.key || e.target.node === payload.key,
    );
    for (const e of connected) {
      const oldEdge = actions.snapshotDraft(e);
      const oldEdgeConfig = snapshotConfig(state, e.key);
      delete state.configs[e.key];
      inverse.push(addEdge({ edge: oldEdge }));
      if (oldEdgeConfig != null)
        inverse.push(setConfig({ key: e.key, config: oldEdgeConfig }));
      targets.push(e.key);
    }
    state.edges = state.edges.filter(
      (e) => e.source.node !== payload.key && e.target.node !== payload.key,
    );
    for (const [key, config] of Object.entries(state.configs)) {
      if (config?.variant !== "group_box") continue;
      const { members } = config;
      if (!members.includes(payload.key)) continue;
      const oldConfig = actions.snapshotDraft(config);
      config.members = members.filter((m) => m !== payload.key);
      inverse.push(setConfig({ key, config: oldConfig }));
      targets.push(key);
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
    const oldConfig = snapshotConfig(state, payload.key);
    state.edges.splice(idx, 1);
    delete state.configs[payload.key];
    const inverse: Action[] = [addEdge({ edge: oldEdge })];
    if (oldConfig != null)
      inverse.push(setConfig({ key: payload.key, config: oldConfig }));
    return { inverse, targets: [payload.key] };
  },
  setConfig: (state, payload) => {
    if (!hasElement(state, payload.key)) return actions.NO_OP_RESULT;
    const prev = snapshotConfig(state, payload.key);
    state.configs[payload.key] = payload.config;
    return {
      inverse: prev != null ? [setConfig({ key: payload.key, config: prev })] : [],
      targets: [payload.key],
    };
  },
  ...actions.createPathHandlers(applyPath),
};

export const reduceAll = createReduceAll(handlers);

export interface SetChangedArgs {
  /** The key of the element. */
  key: string;
  base: ElementConfig;
  next: ElementConfig;
}

/**
 * Writes only the fields of an element's config that differ between base and next, so
 * a concurrent edit to another field of the element survives.
 * @returns the actions to dispatch, with none when nothing differs.
 */
export const setChanged = ({ key, base, next }: SetChangedArgs): Action[] =>
  patch.diff(elementConfigZ, base, next).map((a) => patch.prefix(a, ["configs", key]));

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
