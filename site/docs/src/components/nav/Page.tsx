// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Imported as a file, since a bare import of the side-effect-free tree module is
// dropped with its styles.
import "@synnaxlabs/lyra/dist/tree/Item.css";

import { Button } from "@synnaxlabs/lyra/button";
import { Caret } from "@synnaxlabs/lyra/caret";
import { type CSS } from "@synnaxlabs/lyra/css";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { SECTION_ICONS } from "@/components/nav/icons";
import { ancestry, type PageNavNode } from "@/components/nav/tree";
import { InlineCode } from "@/components/text/InlineCode";
import { normalizeRoute } from "@/util/route";

const ITEM_CLASS = "lyra-tree__item lyra--margin lyra-list__item";
const SELECTED_CLASS = "lyra--selected";

interface Row {
  node: PageNavNode;
  depth: number;
  expanded: boolean;
  hidden: boolean;
}

const idOf = (node: PageNavNode): string => node.href ?? node.key;

// Lists every node in reading order, as Lyra's tree does, so a collapsed node's
// descendants follow it and hide with it.
const rows = (
  nodes: PageNavNode[],
  open: Set<PageNavNode>,
  depth = 0,
  hidden = false,
): Row[] =>
  nodes.flatMap((node) => {
    const expanded = depth === 0 || open.has(node);
    return [
      { node, depth, expanded, hidden },
      ...rows(node.children ?? [], open, depth + 1, hidden || !expanded),
    ];
  });

const caret = (expanded: boolean): ReactElement => (
  <Caret.Animated
    className="lyra-tree__expansion-indicator"
    enabled={expanded}
    enabledLoc="bottom"
    disabledLoc="right"
  />
);

const SECTION_STYLE: CSS.VarProperties = {
  "--pluto-tree-item-offset": "1.5rem",
  textDecoration: "none",
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
    {SECTION_ICONS[idOf(node)]}
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
      className={`${ITEM_CLASS} ${depth !== 0 ? "lyra--show-rules" : ""} ${selected ? SELECTED_CLASS : ""}`}
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
  id?: string;
  nodes: PageNavNode[];
}

/**
 * Renders the reference tree on the server with the sections and the current page's
 * ancestors expanded. The script in `@/components/nav/sidebar` expands nodes and
 * follows page swaps, so the tree never hydrates.
 */
export const Page = ({ id, nodes, currentPage: path }: TreeProps): ReactElement => {
  const currentPage = normalizeRoute(path);
  const open = new Set(ancestry(nodes, currentPage));
  return (
    <div
      id={id}
      className="lyra-tree tree reference-tree styled-scrollbar lyra-list__scroll lyra-flex lyra--full-y"
      role="tree"
    >
      <div className="lyra-list__virtualizer">
        {rows(nodes, open).map((row) =>
          row.depth === 0 && row.node.children != null ? (
            <Section key={idOf(row.node)} {...row} />
          ) : (
            <Item key={idOf(row.node)} {...row} currentPage={currentPage} />
          ),
        )}
      </div>
    </div>
  );
};
