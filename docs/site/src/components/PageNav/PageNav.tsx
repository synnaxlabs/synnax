// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, Logo } from "@synnaxlabs/media";
import { Align } from "@synnaxlabs/pluto";
import { Button } from "@synnaxlabs/pluto/button";
import { Dropdown } from "@synnaxlabs/pluto/dropdown";
import { Tree } from "@synnaxlabs/pluto/tree";
import { type ReactElement } from "react";

import { componentsPages, guidesPages } from "@/pages/_nav";

export type PageNavNode = Tree.Node;

export interface TOCProps {
  currentPage: string;
}

interface ReferenceTreeProps {
  currentPage: string;
}

const Reference = ({ currentPage }: ReferenceTreeProps): ReactElement => {
  let parts = currentPage.split("/").filter((part) => part !== "");
  if (parts.length <= 1) parts = componentsPages.map((p) => p.key);
  if (currentPage === "/guides/") currentPage = "/reference/";
  const treeProps = Tree.use({
    nodes: componentsPages,
    initialExpanded: parts,
    sort: false,
  });
  return (
    <Tree.Tree
      {...treeProps}
      className="tree reference-tree styled-scrollbar"
      virtual={false}
      selected={[currentPage]}
      useMargin
    />
  );
};

const Guides = ({ currentPage }: TOCProps): ReactElement => {
  let parts = currentPage.split("/").filter((part) => part !== "");
  if (parts.length <= 1) parts = guidesPages.map((p) => p.key);
  if (currentPage === "/reference/") currentPage = "/guides/";
  const treeProps = Tree.use({
    nodes: guidesPages,
    initialExpanded: parts,
    sort: false,
  });
  return (
    <Tree.Tree
      {...treeProps}
      className="tree role-tree styled-scrollbar"
      virtual={false}
      selected={[currentPage]}
      useMargin
    />
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
  const { visible, toggle, close } = Dropdown.use({ initialVisible: false });
  return (
    <Dropdown.Dialog
      visible={visible}
      close={close}
      variant="modal"
      location="top"
      className="page-nav-mobile"
    >
      <Button.Icon onClick={toggle} size="large" variant="outlined">
        <Icon.Menu />
      </Button.Icon>
      <Align.Space borderShade={4} bordered rounded className="page-nav-mobile-content">
        <Align.Space
          style={{
            width: "100%",
            padding: "2rem 2rem",
            borderBottom: "var(--pluto-border)",
          }}
          direction="x"
        >
          <Logo variant="title" />
        </Align.Space>
        {tree}
      </Align.Space>
    </Dropdown.Dialog>
  );
};
