// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// The tree reuses the item styles Lyra's tree ships.
import "@synnaxlabs/lyra/tree";

import { Button } from "@synnaxlabs/lyra/button";
import { Caret } from "@synnaxlabs/lyra/caret";
import { type CSS } from "@synnaxlabs/lyra/css";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { InlineCode } from "@/components/text/InlineCode";
import { type Flag } from "@/flags";
import { normalizeRoute } from "@/util/route";

const SECTION_ICONS: Record<string, ReactElement> = {
  concepts: <Icon.Reference />,
  core: <Icon.Core />,
  client: <Icon.Terminal />,
  control: <Icon.Control />,
  console: <Icon.Dashboard />,
  driver: <Icon.Device />,
  pluto: <Icon.Visualize />,
};

export interface PageNavNode {
  key: string;
  name: string;
  href?: string;
  icon?: string;
  /** Hides the node and its children while the flag is off. */
  flag?: Flag;
  children?: PageNavNode[];
}

const ITEM_CLASS = "pluto-tree__item pluto--margin pluto-list__item";
const SELECTED_CLASS = "pluto--selected";

interface Row {
  node: PageNavNode;
  depth: number;
  expanded: boolean;
  hidden: boolean;
}

// Lists every node in reading order, as Lyra's tree does, so a collapsed node's
// descendants follow it and hide with it.
const rows = (
  nodes: PageNavNode[],
  open: Set<string>,
  depth = 0,
  hidden = false,
): Row[] =>
  nodes.flatMap((node) => {
    const expanded = depth === 0 || open.has(node.key);
    return [
      { node, depth, expanded, hidden },
      ...rows(node.children ?? [], open, depth + 1, hidden || !expanded),
    ];
  });

const caret = (expanded: boolean): ReactElement => (
  <Caret.Animated
    className="pluto-tree__expansion-indicator"
    enabled={expanded}
    enabledLoc="bottom"
    disabledLoc="right"
  />
);

const SECTION_STYLE: CSS.VarProperties = {
  "--pluto-tree-item-offset": "1.5rem",
  textDecoration: "none",
  paddingLeft: "0.5rem",
  paddingRight: "0.5rem",
};

const Section = ({ node }: Row): ReactElement => (
  <Button.Button
    el="div"
    variant="text"
    className={`${ITEM_CLASS} page-nav-section-header`}
    role="treeitem"
    aria-level={1}
    aria-expanded
    align="center"
    gap={1.5}
    preventClick
    style={SECTION_STYLE}
  >
    {caret(true)}
    {SECTION_ICONS[node.key]}
    <Text.Text level="p" weight={500}>
      <InlineCode text={node.name} />
    </Text.Text>
  </Button.Button>
);

const Item = ({
  node,
  depth,
  expanded,
  hidden,
  currentPage,
}: Row & PageProps): ReactElement => {
  const style: CSS.VarProperties = {
    textDecoration: "none",
    paddingLeft: "2.5rem",
    paddingRight: "0.5rem",
    "--pluto-tree-item-offset": `${depth * 1.5 + 1.5}rem`,
  };
  const parent = node.children != null;
  const selected = node.href != null && normalizeRoute(node.href) === currentPage;
  return (
    <Button.Button
      el="a"
      variant="text"
      href={node.href}
      className={`${ITEM_CLASS} pluto--show-rules ${selected ? SELECTED_CLASS : ""}`}
      role="treeitem"
      aria-level={depth + 1}
      aria-expanded={parent ? expanded : undefined}
      aria-selected={selected}
      align="center"
      gap="small"
      hidden={hidden}
      style={style}
    >
      {parent && caret(expanded)}
      <Text.Text weight={450}>
        <InlineCode text={node.name} />
      </Text.Text>
    </Button.Button>
  );
};

export interface PageProps {
  currentPage: string;
}

export interface TreeProps extends PageProps {
  nodes: PageNavNode[];
}

/**
 * Renders the reference tree on the server with the sections and the current page's
 * ancestors expanded. The script in `@/components/nav/sidebar` expands nodes and
 * follows page swaps, so the tree never hydrates.
 */
export const Page = ({ nodes, currentPage: path }: TreeProps): ReactElement => {
  const currentPage = normalizeRoute(path);
  const open = new Set(currentPage.split("/").filter((part) => part !== ""));
  return (
    <div
      className="pluto-tree tree reference-tree styled-scrollbar pluto-list__scroll pluto-flex pluto--full-y"
      role="tree"
    >
      <div className="pluto-list__virtualizer">
        {rows(nodes, open).map((row) =>
          row.depth === 0 && row.node.children != null ? (
            <Section key={row.node.key} {...row} />
          ) : (
            <Item key={row.node.key} {...row} currentPage={currentPage} />
          ),
        )}
      </div>
    </div>
  );
};
