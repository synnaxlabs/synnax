// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, type resource } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Access, Arc } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { useSearch } from "@/feature/arc/toolbar/search";
import { RootContextMenu } from "@/feature/arc/tree";
import { Arc as PlatformArc } from "@/platform/arc";
import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { type Nav } from "@/platform/nav";
import { Toolbar } from "@/platform/toolbar";
import { Tree } from "@/platform/tree";

interface EmptyContentProps {
  onCreate: () => void;
}

const EmptyContent = ({ onCreate }: EmptyContentProps) => {
  const hasCreatePermission = Access.useCreateGranted(arc.TYPE_RESOURCE_ID);
  return (
    <Empty.Action
      message="No Arcs"
      action={hasCreatePermission ? "Create Arc automation" : undefined}
      onClick={onCreate}
    />
  );
};

const Content = (): ReactElement => {
  const rootKey = Arc.Directory.useRootKey({});
  return <Loaded root={arc.directory.resourceID(rootKey)} />;
};

interface LoadedProps {
  root: resource.ID;
}

const Loaded = ({ root }: LoadedProps): ReactElement => {
  const create = PlatformArc.useCreate();
  const search = useSearch(root);
  return (
    <Toolbar.Content className={CSS.B("arc-toolbar")}>
      <Toolbar.Header>
        {search.open ? (
          <Tree.SearchField search={search} />
        ) : (
          <>
            <Toolbar.Title>
              <Icon.Arc />
              Arcs
            </Toolbar.Title>
            <Actions search={search} handleCreate={create} />
          </>
        )}
      </Toolbar.Header>
      <Toolbar.Body>
        <Tree.Tree
          root={root}
          emptyContent={<EmptyContent onCreate={create} />}
          rootContextMenu={RootContextMenu}
          search={search}
        />
      </Toolbar.Body>
    </Toolbar.Content>
  );
};

interface ActionsProps {
  search: Tree.Search;
  handleCreate: () => void;
}

const Actions = ({ search, handleCreate }: ActionsProps): ReactElement | null => {
  const hasCreatePermission = Access.useCreateGranted(arc.TYPE_RESOURCE_ID);
  const hasRetrievePermission = Access.useRetrieveGranted(arc.TYPE_RESOURCE_ID);
  if (!hasCreatePermission && !hasRetrievePermission) return null;
  return (
    <Toolbar.Actions>
      <Tree.SearchAction search={search} />
      {hasCreatePermission && (
        <Toolbar.CreateAction tooltip="Create Arc automation" onClick={handleCreate} />
      )}
    </Toolbar.Actions>
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
  useVisible: () => Access.useRetrieveGranted(arc.TYPE_RESOURCE_ID),
};
