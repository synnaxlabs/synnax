// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access } from "@synnaxlabs/client";
import { Access, Icon, Menu as PMenu } from "@synnaxlabs/pluto";

import { Menu } from "@/components";
import { Ontology } from "@/ontology";
import { createUseDelete } from "@/ontology/createUseDelete";
import { createUseRename } from "@/ontology/createUseRename";

const useDelete = createUseDelete({
  type: "Policy",
  query: Access.Policy.useDelete,
  convertKey: String,
});

const useRename = createUseRename({
  query: Access.Policy.useRename,
  ontologyID: access.policy.ontologyID,
  convertKey: String,
});

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection: { ids },
    state,
  } = props;
  const handleDelete = useDelete(props);
  const handleRename = useRename(props);
  const handleSelect = {
    rename: handleRename,
    delete: handleDelete,
  };
  const singleResource = ids.length === 1;
  const resources = state.getResource(ids);
  const hasInternal = resources.some((r) => r.data?.internal === true);
  return (
    <PMenu.Menu onChange={handleSelect} level="small" gap="small">
      {singleResource && !hasInternal && (
        <>
          <Menu.RenameItem />
          <PMenu.Divider />
        </>
      )}
      {!hasInternal && (
        <>
          <Menu.DeleteItem />
          <PMenu.Divider />
        </>
      )}
      {singleResource && (
        <>
          <Ontology.CopyMenuItem {...props} />
          <PMenu.Divider />
        </>
      )}
      <Menu.ReloadConsoleItem />
    </PMenu.Menu>
  );
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: "policy",
  icon: <Icon.Policy />,
  TreeContextMenu,
  hasChildren: false,
};
