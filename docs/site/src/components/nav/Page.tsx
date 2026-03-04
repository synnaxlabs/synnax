// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Logo } from "@synnaxlabs/media";
import { Component, Flex, Icon, List, Text } from "@synnaxlabs/pluto";
import { Tree } from "@synnaxlabs/pluto/tree";
import { type CSSProperties, type ReactElement, useEffect, useState } from "react";

import { REFERENCE_PAGES } from "@/pages/_nav";

interface InternalTreeProps {
  currentPage: string;
}

const SECTION_ICONS: Record<string, ReactElement> = {
  concepts: <Icon.Reference />,
  core: <Icon.Cluster />,
  client: <Icon.Terminal />,
  control: <Icon.Control />,
  console: <Icon.Dashboard />,
  driver: <Icon.Device />,
  pluto: <Icon.Visualize />,
};

export type PageNavNode = Omit<Tree.Node<string>, "children"> & {
  name: string;
  href?: string;
  icon?: string;
  children?: PageNavNode[];
};

export interface TOCProps {
  currentPage?: string;
}

const useCurrentPage = (initialPage?: string): string => {
  const [currentPage, setCurrentPage] = useState(
    () =>
      initialPage ?? (typeof window !== "undefined" ? window.location.pathname : "/"),
  );

  useEffect(() => {
    const update = () => setCurrentPage(window.location.pathname);
    window.addEventListener("popstate", update);
    document.addEventListener("astro:after-swap", update);
    return () => {
      window.removeEventListener("popstate", update);
      document.removeEventListener("astro:after-swap", update);
    };
  }, []);

  return currentPage;
};

const Item = ({ translate: _, ...props }: Tree.ItemRenderProps<string>) => {
  const { itemKey, index } = props;
  const item = List.useItem<string, PageNavNode>(itemKey);
  const { depth, hasChildren } = Tree.useContext("Item")[index];
  if (item == null) return null;

  const isSection = depth === 0 && hasChildren;

  if (isSection)
    return (
      <Tree.Item<string, "div">
        {...props}
        className="page-nav-section-header"
        style={{
          textDecoration: "none",
          paddingLeft: "0.5rem",
          paddingRight: "0.5rem",
        }}
        offsetMultiplier={0}
        el="div"
        useMargin
        gap={1.5}
        preventClick
      >
        {SECTION_ICONS[itemKey]}
        <Text.Text level="p" weight={500}>
          {item.name}
        </Text.Text>
      </Tree.Item>
    );

  const offset = depth * 1.5 + 1.5;

  return (
    <Tree.Item<string, "a">
      {...props}
      style={
        {
          textDecoration: "none",
          paddingLeft: "2.5rem",
          paddingRight: "0.5rem",
          "--pluto-tree-item-offset": `${offset}rem`,
        } as CSSProperties
      }
      el="a"
      href={item.href}
      useMargin
      propagateClick
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

const REFERENCE_SECTION_KEYS = REFERENCE_PAGES.filter((p) => p.children != null).map(
  (p) => p.key,
);

const Reference = ({ currentPage }: InternalTreeProps): ReactElement => {
  let parts = currentPage.split("/").filter((part) => part !== "");
  if (parts.length <= 1) parts = REFERENCE_PAGES.map((p) => p.key);
  const referenceData = flatten(REFERENCE_PAGES);
  const nodesStore = List.useMapData({ initialData: referenceData });
  const treeProps = Tree.use({
    nodes: REFERENCE_PAGES,
    initialExpanded: [...parts, ...REFERENCE_SECTION_KEYS],
    onExpand: ({ action, clicked }) => {
      if (action === "contract" && REFERENCE_SECTION_KEYS.includes(clicked))
        treeProps.expand(clicked);
    },
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

export const Page = ({ currentPage: initialPage }: TOCProps): ReactElement | null => {
  const currentPage = useCurrentPage(initialPage);
  return <Reference currentPage={currentPage} />;
};

export const PageMobile = ({ currentPage: initialPage }: TOCProps): ReactElement => {
  const currentPage = useCurrentPage(initialPage);
  const [open, setOpen] = useState(false);

  useEffect(() => {
    if (open) document.body.classList.add("mobile-menu-open");
    else document.body.classList.remove("mobile-menu-open");
    return () => document.body.classList.remove("mobile-menu-open");
  }, [open]);

  return (
    <>
      <button
        className="mobile-menu-btn"
        aria-label="Open menu"
        onClick={() => setOpen(true)}
      >
        <span className="mobile-menu-icon" />
      </button>
      {open && (
        <div
          className="mobile-overlay mobile-overlay--open"
          onClick={() => setOpen(false)}
        />
      )}
      <nav className={`mobile-drawer ${open ? "mobile-drawer--open" : ""}`}>
        <div className="mobile-drawer-header">
          <a href="https://synnaxlabs.com" className="logo-link">
            <Logo variant="title" />
          </a>
          <button
            className="mobile-close-btn"
            aria-label="Close menu"
            onClick={() => setOpen(false)}
          >
            <span className="mobile-close-icon" />
          </button>
        </div>
        <div className="mobile-drawer-links">
          <a
            href="/reference/"
            className="mobile-drawer-link"
            onClick={() => setOpen(false)}
          >
            Reference
          </a>
          <a
            href="/blog/"
            className="mobile-drawer-link"
            onClick={() => setOpen(false)}
          >
            Blog
          </a>
          <a
            href="/releases/"
            className="mobile-drawer-link"
            onClick={() => setOpen(false)}
          >
            Releases
          </a>
          <div className="mobile-drawer-divider" />
          <span className="mobile-drawer-section-label">Reference</span>
        </div>
        <div className="mobile-drawer-tree">
          <Reference currentPage={currentPage} />
        </div>
      </nav>
    </>
  );
};
