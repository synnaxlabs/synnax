// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { deep } from "@synnaxlabs/x";

import {
  type ChannelName,
  diffProperties,
  humanize,
  type Kind,
  type Property,
} from "@/feature/vcs/diff/change/property";

/** A symbol or connection of a schematic that differs between two versions. */
export interface Element {
  key: string;
  variant: "symbol" | "connection";
  label: string;
  kind: Kind;
  /** properties lists what changed. Empty for an added or removed element. */
  properties: Property[];
}

/** How two versions of a schematic differ. */
export interface Elements {
  elements: Element[];
  /** document lists the changes to the schematic itself, such as its name. */
  document: Property[];
  /** before maps each changed or removed symbol of the old version to its kind. */
  before: Map<string, Kind>;
  /** after maps each changed or added symbol of the new version to its kind. */
  after: Map<string, Kind>;
}

export interface ElementsOptions {
  channelName?: ChannelName;
  /** symbolName names a symbol that has no label, such as "Valve". */
  symbolName: (config: schematic.ElementConfig) => string;
}

const labelOf = (
  config: schematic.ElementConfig | undefined,
  symbolName: ElementsOptions["symbolName"],
): string => {
  if (config == null) return "Symbol";
  if ("label" in config) {
    const { label } = config;
    if (label != null && typeof label === "object" && "label" in label) {
      const text = label.label;
      if (typeof text === "string" && text.trim().length > 0) return text;
    }
  }
  return symbolName(config);
};

const formatPosition = ({ x, y }: { x: number; y: number }): string =>
  `${Math.round(x)}, ${Math.round(y)}`;

const NODE_OMIT = ["key", "position"];
// Moving a symbol reroutes the pipes attached to it, so routes read as noise.
const EDGE_CONFIG_OMIT = ["segments"];
const DOCUMENT_OMIT = ["key", "nodes", "edges", "configs"];

/**
 * Compares two versions of a schematic symbol by symbol and connection by connection.
 * Symbols match by key. A symbol whose position or config changed is changed.
 */
export const diffElements = (
  before: schematic.Schematic,
  after: schematic.Schematic,
  { channelName, symbolName }: ElementsOptions,
): Elements => {
  const elements: Element[] = [];
  const beforeKinds = new Map<string, Kind>();
  const afterKinds = new Map<string, Kind>();
  const beforeNodes = new Map(before.nodes.map((n) => [n.key, n]));
  const afterNodes = new Map(after.nodes.map((n) => [n.key, n]));

  after.nodes.forEach((node) => {
    const prev = beforeNodes.get(node.key);
    const config = after.configs[node.key];
    const label = labelOf(config, symbolName);
    if (prev == null) {
      afterKinds.set(node.key, "added");
      elements.push({
        key: node.key,
        variant: "symbol",
        label,
        kind: "added",
        properties: [],
      });
      return;
    }
    const properties: Property[] = [];
    if (!deep.equal(prev.position, node.position))
      properties.push({
        path: ["Position"],
        kind: "changed",
        before: formatPosition(prev.position),
        after: formatPosition(node.position),
      });
    properties.push(
      ...diffProperties(prev, node, { channelName, omit: NODE_OMIT }),
      ...diffProperties(before.configs[node.key], config, { channelName }),
    );
    if (properties.length === 0) return;
    beforeKinds.set(node.key, "changed");
    afterKinds.set(node.key, "changed");
    elements.push({
      key: node.key,
      variant: "symbol",
      label,
      kind: "changed",
      properties,
    });
  });

  before.nodes.forEach((node) => {
    if (afterNodes.has(node.key)) return;
    beforeKinds.set(node.key, "removed");
    elements.push({
      key: node.key,
      variant: "symbol",
      label: labelOf(before.configs[node.key], symbolName),
      kind: "removed",
      properties: [],
    });
  });

  const edgeLabel = (doc: schematic.Schematic, edge: schematic.Edge): string => {
    const config = doc.configs[edge.key];
    const name = config == null ? "Connection" : humanize(config.variant);
    const source = labelOf(doc.configs[edge.source.node], symbolName);
    const target = labelOf(doc.configs[edge.target.node], symbolName);
    return `${name} from ${source} to ${target}`;
  };
  const beforeEdges = new Map(before.edges.map((e) => [e.key, e]));
  const afterEdges = new Set(after.edges.map((e) => e.key));
  after.edges.forEach((edge) => {
    const prev = beforeEdges.get(edge.key);
    const label = edgeLabel(after, edge);
    if (prev == null) {
      elements.push({
        key: edge.key,
        variant: "connection",
        label,
        kind: "added",
        properties: [],
      });
      return;
    }
    const properties = [
      ...diffProperties(prev, edge, { channelName, omit: ["key"] }),
      ...diffProperties(before.configs[edge.key], after.configs[edge.key], {
        channelName,
        omit: EDGE_CONFIG_OMIT,
      }),
    ];
    if (properties.length === 0) return;
    elements.push({
      key: edge.key,
      variant: "connection",
      label,
      kind: "changed",
      properties,
    });
  });
  before.edges.forEach((edge) => {
    if (afterEdges.has(edge.key)) return;
    elements.push({
      key: edge.key,
      variant: "connection",
      label: edgeLabel(before, edge),
      kind: "removed",
      properties: [],
    });
  });

  return {
    elements,
    document: diffProperties(before, after, { channelName, omit: DOCUMENT_OMIT }),
    before: beforeKinds,
    after: afterKinds,
  };
};
