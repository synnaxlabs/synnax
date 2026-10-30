// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Access, Ontology } from "@synnaxlabs/pluto";
import { type ReactElement, useState } from "react";

import { Explorer } from "@/feature/arc/explorer";
import { RootContextMenu } from "@/feature/arc/tree";
import { Arc } from "@/platform/arc";
import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { type Nav } from "@/platform/nav";
import { Toolbar } from "@/platform/toolbar";
import { Tree } from "@/platform/tree";

interface EmptyContentProps {
  onCreate: () => void;
}

const EmptyContent = ({ onCreate }: EmptyContentProps) => {
  const hasCreatePermission = Access.useCreateGranted(arc.TYPE_ONTOLOGY_ID);
  return (
    <Empty.Action
      message="No Arcs"
      action={hasCreatePermission ? "Create Arc automation" : undefined}
      onClick={onCreate}
    />
  );
};

const Content = (): ReactElement => {
  const groupID = Ontology.useRootChildID({ name: "Arcs" });
  const create = Arc.useCreate();
  const openExplorer = Explorer.useOpenTab();
  const hasCreatePermission = Access.useCreateGranted(arc.TYPE_ONTOLOGY_ID);
  const hasRetrievePermission = Access.useRetrieveGranted(arc.TYPE_ONTOLOGY_ID);
  const [searching, setSearching] = useState(false);
  return (
    <Toolbar.Content className={CSS.B("arc-toolbar")}>
      <Toolbar.Header>
        <Toolbar.Title>
          <Icon.Arc />
          Arcs
        </Toolbar.Title>
        <Toolbar.Actions>
          <Toolbar.SearchAction value={searching} onChange={setSearching} />
          {hasRetrievePermission && (
            <Toolbar.Action tooltip="Open Arc explorer" onClick={openExplorer}>
              <Icon.Explore />
            </Toolbar.Action>
          )}
          {hasCreatePermission && (
            <Toolbar.Action
              tooltip="Create Arc automation"
              onClick={create}
              variant="filled"
            >
              <Icon.Add />
            </Toolbar.Action>
          )}
        </Toolbar.Actions>
      </Toolbar.Header>
      <Toolbar.Body>
        <Tree.Tree
          root={groupID}
          search={searching}
          emptyContent={<EmptyContent onCreate={create} />}
          rootContextMenu={RootContextMenu}
        />
      </Toolbar.Body>
    </Toolbar.Content>
  );
};

export const TOOLBAR: Nav.Toolbar = {
  key: "arc",
  icon: <Icon.Arc />,
  content: <Content />,
  trigger: ["A"],
  tooltip: "Arcs",
  sizeBounds: { lower: 225, upper: 400 },
  initialSize: 300,
  useVisible: () => Access.useRetrieveGranted(arc.TYPE_ONTOLOGY_ID),
};
