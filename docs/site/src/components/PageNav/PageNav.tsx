// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Logo } from "@synnaxlabs/media";
import { Component, Dialog, Flex, Icon, List, Text } from "@synnaxlabs/pluto";
import { Tree } from "@synnaxlabs/pluto/tree";
import { type ReactElement } from "react";

import { GUIDES_PAGES, REFERENCE_PAGES } from "@/pages/_nav";

export type PageNavNode = Omit<Tree.Node<string>, "children"> & {
  name: string;
  href?: string;
  children?: PageNavNode[];
};

export interface TOCProps {
  currentPage: string;
}

interface ReferenceTreeProps {
  currentPage: string;
}

const Item = ({ translate: _, ...props }: Tree.ItemRenderProps<string>) => {
  const { itemKey } = props;
  const item = List.useItem<string, PageNavNode>(itemKey);
  if (item == null) return null;
  return (
    <Tree.Item<string, "a">
      {...props}
      style={{
        textDecoration: "none",
        paddingLeft: "2.5rem",
        paddingRight: "0.5rem",
      }}
      offsetMultiplier={3.5}
      el="a"
      href={item.href}
      useMargin
    >
      <Text.Text weight={450}>{item.name}</Text.Text>
    </Tree.Item>
  );
};

const item = Component.renderProp(Item);

const flatten = (nodes: PageNavNode[]): PageNavNode[] => {
  const flattened: PageNavNode[] = [];
  nodes.forEach((node) => {
    flattened.push(node);
    if (node.children != null) flattened.push(...flatten(node.children));
  });
  return flattened;
};

const REFERENCE_DATA = flatten(REFERENCE_PAGES);

const Reference = ({ currentPage }: ReferenceTreeProps): ReactElement => {
  let parts = currentPage.split("/").filter((part) => part !== "");
  if (parts.length <= 1) parts = REFERENCE_PAGES.map((p) => p.key);
  if (currentPage === "/guides/") currentPage = "/reference/";
  const nodesStore = List.useMapData({ initialData: REFERENCE_DATA });
  const treeProps = Tree.use({
    nodes: REFERENCE_PAGES,
    initialExpanded: parts,
  });
  return (
    <Tree.Tree
      {...treeProps}
      className="tree reference-tree styled-scrollbar"
      virtual={false}
      selected={[currentPage]}
      getItem={nodesStore.getItem}
      subscribe={nodesStore.subscribe}
    >
      {item}
    </Tree.Tree>
  );
};

const GUIDES_DATA = flatten(GUIDES_PAGES);

const Guides = ({ currentPage }: TOCProps): ReactElement => {
  let parts = currentPage.split("/").filter((part) => part !== "");
  if (parts.length <= 1) parts = GUIDES_PAGES.map((p) => p.key);
  if (currentPage === "/reference/") currentPage = "/guides/";
  const nodesStore = List.useMapData({ initialData: GUIDES_DATA });
  const treeProps = Tree.use({
    nodes: GUIDES_PAGES,
    initialExpanded: parts,
    sort: (a, b) => a.key.localeCompare(b.key),
  });
  return (
    <Tree.Tree
      {...treeProps}
      className="tree role-tree styled-scrollbar"
      virtual={false}
      selected={[currentPage]}
      getItem={nodesStore.getItem}
      subscribe={nodesStore.subscribe}
    >
      {item}
    </Tree.Tree>
  );
};

export const PageNav = ({ currentPage }: TOCProps): ReactElement | null => {
  const selectedTab = currentPage.split("/").filter((part) => part !== "")[0];
  let tree = <Reference currentPage={currentPage} />;
  if (selectedTab === "guides") tree = <Guides currentPage={currentPage} />;
  return tree;
};

export const PageNavMobile = ({ currentPage }: TOCProps): ReactElement => {
  const selectedTab = currentPage.split("/").filter((part) => part !== "")[0];
  let tree = <Reference currentPage={currentPage} />;
  if (selectedTab === "guides") tree = <Guides currentPage={currentPage} />;
  return (
    <Dialog.Frame variant="modal" location="top" className="page-nav-mobile">
      <Dialog.Trigger size="large" variant="outlined">
        <Icon.Menu />
      </Dialog.Trigger>
      <Dialog.Dialog>
        <Flex.Box
          borderColor={5}
          background={0}
          bordered
          rounded
          className="page-nav-mobile-content"
        >
          <Flex.Box
            style={{
              width: "100%",
              padding: "2rem 2rem",
              borderBottom: "var(--pluto-border)",
            }}
            direction="x"
          >
            <Logo variant="title" />
          </Flex.Box>
          {tree}
        </Flex.Box>
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};
