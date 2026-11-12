// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type crdt, patch, type record } from "@synnaxlabs/x";
import { type Draft } from "immer";

import { actions } from "@/actions";
import {
  type Action,
  addEdge,
  createReduceAll,
  type HandlerResult,
  type Handlers,
  reconnectEdge,
  removeEdge,
  removeNode,
  setDirectory,
  setNode,
  setNodeInputs,
  setNodePosition,
} from "@/arc/actions.gen";
import { type ir } from "@/arc/ir";
import {
  type Arc,
  arcZ,
  type GraphSource,
  graphSourceZ,
  type TextSource,
} from "@/arc/types.gen";

const idToString = (id: crdt.ID): string => `${id.replica}:${id.counter}`;

const sameEndpoints = (a: ir.Edge, source: ir.Handle, target: ir.Handle): boolean =>
  a.source.node === source.node &&
  a.source.param === source.param &&
  a.target.node === target.node &&
  a.target.param === target.param;

const wrongMode = (state: Draft<Arc>, edit: string): Error =>
  new Error(
    `${state.name} is a ${state.source.mode} Arc and cannot take a ${edit} edit`,
  );

// graphSourceOf returns the source of a graph-mode Arc, throwing for a text-mode one.
const graphSourceOf = (state: Draft<Arc>): Draft<GraphSource> => {
  if (state.source.mode !== "graph") throw wrongMode(state, "graph");
  return state.source;
};

const graphOf = (state: Draft<Arc>): Draft<GraphSource["graph"]> =>
  graphSourceOf(state).graph;

// textOf returns the text of a text-mode Arc, throwing for a graph-mode one.
const textOf = (state: Draft<Arc>): Draft<TextSource["text"]> => {
  if (state.source.mode !== "text") throw wrongMode(state, "text");
  return state.source.text;
};

// isInputsPath reports whether a path action edits the graph's node inputs, whose paths
// start at the Arc's source. Other path actions edit the fields of the Arc.
const isInputsPath = (action: patch.Action): boolean =>
  patch.pathOf(action)[0] === "graph";

const applyPath = (state: Draft<Arc>, action: patch.Action): HandlerResult =>
  isInputsPath(action)
    ? actions.applyEntryPath(
        graphSourceOf(state),
        action,
        ["graph", "inputs"],
        graphSourceZ,
      )
    : actions.applyPath(state, action, arcZ);

const handlers: Handlers = {
  create: (state, payload) => {
    Object.assign(state, payload.arc);
    return { inverse: [], targets: [payload.arc.key] };
  },

  setNode: (state, payload) => {
    const graph = graphOf(state);
    const idx = graph.nodes.findIndex((n) => n.key === payload.node.key);
    if (idx === -1) {
      graph.nodes.push(payload.node);
      return {
        inverse: [removeNode({ key: payload.node.key })],
        targets: [payload.node.key],
      };
    }
    const oldNode = actions.snapshotDraft(graph.nodes[idx]);
    graph.nodes[idx] = payload.node;
    return { inverse: [setNode({ node: oldNode })], targets: [payload.node.key] };
  },
  setNodePosition: (state, payload) => {
    const graph = graphOf(state);
    const node = graph.nodes.find((n) => n.key === payload.key);
    if (node == null) return actions.NO_OP_RESULT;
    const oldPosition = { x: node.position.x, y: node.position.y };
    node.position = payload.position;
    return {
      inverse: [setNodePosition({ key: payload.key, position: oldPosition })],
      targets: [payload.key],
    };
  },
  // The inverse of SetNodeInputs is imperfect for keys the action newly introduces:
  // SetNodeInputs only merges, so it cannot remove keys that did not previously exist.
  // The inverse here restores values for keys that DID exist before the merge; keys
  // added by the action remain on undo as phantom fields. A future ReplaceNodeInputs
  // action can close the gap by enabling wholesale replacement.
  setNodeInputs: (state, payload) => {
    const graph = graphOf(state);
    const existingRaw = graph.inputs[payload.key];
    if (existingRaw == null) {
      graph.inputs[payload.key] = payload.inputs;
      return { inverse: [], targets: [payload.key] };
    }
    const existing = actions.snapshotDraft(existingRaw);
    const restoreFields: record.Unknown = {};
    for (const k of Object.keys(payload.inputs))
      if (existing[k] !== undefined) restoreFields[k] = existing[k];
    graph.inputs[payload.key] = { ...existing, ...payload.inputs };
    if (Object.keys(restoreFields).length === 0)
      return { inverse: [], targets: [payload.key] };
    return {
      inverse: [setNodeInputs({ key: payload.key, inputs: restoreFields })],
      targets: [payload.key],
    };
  },
  removeNode: (state, payload) => {
    const graph = graphOf(state);
    const idx = graph.nodes.findIndex((n) => n.key === payload.key);
    if (idx === -1) return actions.NO_OP_RESULT;
    const oldNode = actions.snapshotDraft(graph.nodes[idx]);
    const oldInputs = actions.snapshotDraft(graph.inputs[payload.key]);
    const removedEdges = graph.edges
      .filter((e) => e.source.node === payload.key || e.target.node === payload.key)
      .map((e) => actions.snapshotDraft(e));
    graph.nodes.splice(idx, 1);
    delete graph.inputs[payload.key];
    graph.edges = graph.edges.filter(
      (e) => e.source.node !== payload.key && e.target.node !== payload.key,
    );
    const inverse: Action[] = [setNode({ node: oldNode })];
    if (oldInputs != null)
      inverse.push(setNodeInputs({ key: payload.key, inputs: oldInputs }));
    inverse.push(...removedEdges.map((e) => addEdge({ edge: e })));
    return { inverse, targets: [payload.key] };
  },
  addEdge: (state, payload) => {
    const graph = graphOf(state);
    const { edge } = payload;
    if (graph.edges.some((e) => sameEndpoints(e, edge.source, edge.target)))
      return actions.NO_OP_RESULT;
    graph.edges.push(edge);
    return { inverse: [removeEdge({ key: edge.key })], targets: [edge.key] };
  },
  removeEdge: (state, payload) => {
    const graph = graphOf(state);
    const idx = graph.edges.findIndex((e) => e.key === payload.key);
    if (idx === -1) return actions.NO_OP_RESULT;
    const oldEdge = actions.snapshotDraft(graph.edges[idx]);
    graph.edges.splice(idx, 1);
    return { inverse: [addEdge({ edge: oldEdge })], targets: [payload.key] };
  },
  reconnectEdge: (state, payload) => {
    const graph = graphOf(state);
    const edge = graph.edges.find((e) => e.key === payload.key);
    if (edge == null) return actions.NO_OP_RESULT;
    const oldSource = actions.snapshotDraft(edge.source);
    const oldTarget = actions.snapshotDraft(edge.target);
    edge.source = payload.source;
    edge.target = payload.target;
    return {
      inverse: [
        reconnectEdge({ key: payload.key, source: oldSource, target: oldTarget }),
      ],
      targets: [payload.key],
    };
  },
  // Undo/redo is handled the editor, which emits the corresponding
  // insertChar, deleteChar actions.
  insertChar: (state, payload) => {
    const text = textOf(state);
    text.doc.inserts.push(payload);
    return { inverse: [], targets: [] };
  },
  deleteChar: (state, payload) => {
    const text = textOf(state);
    text.doc.deletes.push(payload);
    return { inverse: [], targets: [] };
  },
  // Server-emitted maintenance: drop the insert and delete ops of already-deleted
  // characters. The characters are tombstoned and invisible, so removing their ops does
  // not change the materialized text.
  forgetChars: (state, payload) => {
    if (payload.ids.length === 0) return { inverse: [], targets: [] };
    const text = textOf(state);
    const forget = new Set(payload.ids.map(idToString));
    text.doc.inserts = text.doc.inserts.filter((i) => !forget.has(idToString(i.id)));
    text.doc.deletes = text.doc.deletes.filter((d) => !forget.has(idToString(d.id)));
    return { inverse: [], targets: [] };
  },
  setDirectory: (state, payload) => {
    const oldDirectory = state.directory;
    state.directory = payload.directory;
    return {
      inverse: oldDirectory == null ? [] : [setDirectory({ directory: oldDirectory })],
      targets: [state.key],
    };
  },
  ...actions.createPathHandlers(applyPath),
};

export const reduceAll = createReduceAll(handlers);

/**
 * Converts a path action from the camelCase form of the cached Arc into the snake_case
 * form that the Core stores. Other actions keep their form.
 */
export const encode = (state: Arc, action: Action): Action => {
  switch (action.type) {
    case "set":
    case "remove":
    case "insert":
    case "move":
      return isInputsPath(action)
        ? patch.camelToSnake(action, { schema: graphSourceZ, data: state.source })
        : patch.camelToSnake(action, { schema: arcZ, data: state });
    default:
      return action;
  }
};

// createOf hands the dispatch controller the document carried by a create
// action so frames for never-cached documents ingest instead of drop.
export const createOf = (action: Action) =>
  action.type === "create" ? action.create.arc : undefined;

// isUndoable reports whether an action should push onto the undo stack. Graph mutations
// are user-driven and undoable; collaborative text edits are not (they carry no inverse
// and belong to the CRDT, not the graph undo stack), nor is the server's tombstone
// reclamation.
export const isUndoable = (action: Action): boolean =>
  action.type !== "insert_char" &&
  action.type !== "delete_char" &&
  action.type !== "forget_chars";

// kindOf classifies an action batch for the undo coalesce window. A
// node drag dispatches a stream of set_node_position actions for a single
// gesture; classifying them all as "move" collapses them into one undoable.
export const kindOf = (actions: Action[]): string => {
  if (actions.length === 0) return "default";
  const hasMove = actions.some((a) => a.type === "set_node_position");
  const onlyMove = actions.every((a) => a.type === "set_node_position");
  if (hasMove && onlyMove) return "move";
  if (actions.length === 1) return actions[0].type;
  return "transaction";
};
