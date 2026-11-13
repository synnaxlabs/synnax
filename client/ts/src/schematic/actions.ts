// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep, patch } from "@synnaxlabs/x";
import { type Draft } from "immer";

import { actions } from "@/actions";
import {
  type Action,
  addEdge,
  clearEdgeStrokeColor,
  createReduceAll,
  type HandlerResult,
  type Handlers,
  removeEdge,
  removeNode,
  setConfig,
  setEdge,
  setEdgeSegments,
  setEdgeStyle,
  type SetEdgeStylePayload,
  setNode,
  setNodePosition,
} from "@/schematic/actions.gen";
import {
  type Edge,
  type NodeConfig,
  nodeConfigZ,
  type Schematic,
  schematicZ,
} from "@/schematic/types.gen";

const CONFIGS_FIELD = "configs";

const hasNode = (state: Draft<Schematic>, key: string): boolean =>
  state.nodes.some((n) => n.key === key);

const hasEdge = (state: Draft<Schematic>, key: string): boolean =>
  state.edges.some((e) => e.key === key);

// An action on the config of a node that does not exist does nothing.
const applyPath = (state: Draft<Schematic>, action: patch.Action): HandlerResult => {
  const path = patch.pathOf(action);
  if (path[0] !== CONFIGS_FIELD) return actions.applyPath(state, action, schematicZ);
  if (path.length > 1 && !hasNode(state, path[1])) return actions.NO_OP_RESULT;
  return actions.applyEntryPath(state, action, [CONFIGS_FIELD], schematicZ);
};

const snapshotConfig = (
  state: Draft<Schematic>,
  key: string,
): NodeConfig | undefined => {
  const config = state.configs[key];
  return config != null ? actions.snapshotDraft(config) : undefined;
};

const isConfigEdit = (action: Action): boolean => {
  switch (action.type) {
    case "set_config":
    case "set_edge":
    case "set_edge_segments":
    case "set_edge_style":
    case "clear_edge_stroke_color":
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
      inverse.push(addEdge({ edge: actions.snapshotDraft(e) }));
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
    state.edges.splice(idx, 1);
    return { inverse: [addEdge({ edge: oldEdge })], targets: [payload.key] };
  },
  setEdge: (state, payload) => {
    const { key, source, target } = payload.edge;
    if (!hasNode(state, source.node) || !hasNode(state, target.node))
      return actions.NO_OP_RESULT;
    const idx = state.edges.findIndex((e) => e.key === key);
    if (idx === -1) {
      state.edges.push(payload.edge);
      return { inverse: [removeEdge({ key })], targets: [key] };
    }
    const oldEdge = actions.snapshotDraft(state.edges[idx]);
    state.edges[idx] = payload.edge;
    return { inverse: [setEdge({ edge: oldEdge })], targets: [key] };
  },
  setEdgeSegments: (state, payload) => {
    const edge = state.edges.find((e) => e.key === payload.key);
    if (edge == null) return actions.NO_OP_RESULT;
    const segments = actions.snapshotDraft(edge.segments);
    edge.segments = payload.segments;
    return {
      inverse: [setEdgeSegments({ key: payload.key, segments })],
      targets: [payload.key],
    };
  },
  setEdgeStyle: (state, payload) => {
    const edge = state.edges.find((e) => e.key === payload.key);
    if (edge == null) return actions.NO_OP_RESULT;
    const old = actions.snapshotDraft(edge);
    const { variant, strokeColor, strokeWidth } = payload;
    if (variant != null) edge.variant = variant;
    if (strokeColor != null) edge.strokeColor = strokeColor;
    if (strokeWidth != null) edge.strokeWidth = strokeWidth;
    const inverse: Action[] = [
      setEdgeStyle({
        key: payload.key,
        variant: variant != null ? old.variant : undefined,
        strokeColor: strokeColor != null ? old.strokeColor : undefined,
        strokeWidth: strokeWidth != null ? old.strokeWidth : undefined,
      }),
    ];
    // A style set cannot remove a color, so undoing a first color clears it.
    if (strokeColor != null && old.strokeColor == null)
      inverse.push(clearEdgeStrokeColor({ key: payload.key }));
    return { inverse, targets: [payload.key] };
  },
  clearEdgeStrokeColor: (state, payload) => {
    const edge = state.edges.find((e) => e.key === payload.key);
    if (edge?.strokeColor == null) return actions.NO_OP_RESULT;
    const strokeColor = actions.snapshotDraft(edge.strokeColor);
    delete edge.strokeColor;
    return {
      inverse: [setEdgeStyle({ key: payload.key, strokeColor })],
      targets: [payload.key],
    };
  },
  setConfig: (state, payload) => {
    if (!hasNode(state, payload.key)) return actions.NO_OP_RESULT;
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
  /** The key of the node. */
  key: string;
  base: NodeConfig;
  next: NodeConfig;
}

/**
 * Writes only the fields of a node's config that differ between base and next, so a
 * concurrent edit to another field of the node survives.
 * @returns the actions to dispatch, with none when nothing differs.
 */
export const setChanged = ({ key, base, next }: SetChangedArgs): Action[] =>
  patch.diff(nodeConfigZ, base, next).map((a) => patch.prefix(a, ["configs", key]));

export interface SetEdgeChangedArgs {
  base: Edge;
  next: Edge;
}

/**
 * Writes only the fields of an edge that differ between base and next, so a concurrent
 * edit to another field of the edge survives and a deleted edge stays deleted.
 * @returns the actions to dispatch, with none when nothing differs.
 */
export const setEdgeChanged = ({ base, next }: SetEdgeChangedArgs): Action[] => {
  const { key } = next;
  const out: Action[] = [];
  if (base.strokeColor != null && next.strokeColor == null)
    out.push(clearEdgeStrokeColor({ key }));
  const style: SetEdgeStylePayload = { key };
  if (next.variant !== base.variant) style.variant = next.variant;
  if (next.strokeWidth !== base.strokeWidth) style.strokeWidth = next.strokeWidth;
  if (next.strokeColor != null && !deep.equal(next.strokeColor, base.strokeColor))
    style.strokeColor = next.strokeColor;
  if (Object.keys(style).length > 1) out.push(setEdgeStyle(style));
  if (!deep.equal(next.segments, base.segments))
    out.push(setEdgeSegments({ key, segments: next.segments }));
  return out;
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
  // A drag dispatches a stream of `set_node_position` per frame, plus edge sets
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
