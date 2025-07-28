// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/index.css";
import "@/main.css";

import { type ReactElement } from "react";
import ReactDOM from "react-dom/client";

import { Button } from "@/button";
import { Component } from "@/component";
import { Icon } from "@/icon";
import { Pluto } from "@/pluto";
import { Tree } from "@/tree";

const nodes: Tree.Node[] = [
  {
    key: "1",
    children: [
      {
        key: "1.1",
      },
    ],
  },
];

const treeItem = Component.renderProp((props: Tree.ItemProps<string>) => (
  <Tree.Item {...props}>
    {/* <Text.Text level="p">{data[props.itemKey].name}</Text.Text> */}
  </Tree.Item>
));

const Content = () => {
  const treeProps = Tree.use({
    nodes,
  });
  return <Tree.Tree {...treeProps}>{treeItem}</Tree.Tree>;
};

const Main = (): ReactElement => (
  <Pluto.Provider
    connParams={{
      host: "localhost",
      port: 9090,
      username: "synnax",
      password: "seldon",
    }}
  >
    <Button.Button startIcon={<Icon.InProgress />} gap="tiny" variant="text">
      Hello
    </Button.Button>
  </Pluto.Provider>
);

ReactDOM.createRoot(document.getElementById("root")!).render(<Main />);
