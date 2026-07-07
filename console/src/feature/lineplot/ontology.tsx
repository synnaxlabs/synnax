// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot, ontology } from "@synnaxlabs/client";
import { Access, Icon, LinePlot as Base, Menu, Mosaic } from "@synnaxlabs/pluto";
import { array, strings } from "@synnaxlabs/x";

import { useExport } from "@/feature/lineplot/export";
import { Cluster } from "@/platform/cluster";
import { ContextMenu } from "@/platform/context-menu";
import { Export } from "@/platform/export";
import { Group } from "@/platform/group";
import { Link } from "@/platform/link";
import { Ontology } from "@/platform/ontology";
import { Session } from "@/session";

const useDelete = Ontology.createUseDelete({
  type: "Line Plot",
  icon: "LinePlot",
  query: Base.useDelete,
  convertKey: String,
  beforeUpdate: async ({ data, store }) => {
    store.dispatch(Session.LinePlot.remove({ keys: array.toArray(data) }));
    return data;
  },
});

const useRename = Ontology.createUseRename({
  query: Base.useRename,
  ontologyID: lineplot.ontologyID,
  convertKey: String,
});

const TreeContextMenu: Ontology.TreeContextMenu = (props) => {
  const {
    selection: { ids, rootID },
    state: { getResource, shape },
  } = props;
  const handleDelete = useDelete(props);
  const handleLink = Cluster.useCopyLinkToClipboard();
  const handleExport = useExport();
  const rename = useRename(props);
  const group = Group.useCreateFromSelection();
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const firstID = ids[0];
  const isSingle = ids.length === 1;
  const first = getResource(firstID);
  return (
    <ContextMenu.Menu>
      {hasUpdatePermission && (
        <>
          {isSingle && (
            <>
              <ContextMenu.RenameItem onClick={rename} />
              <Menu.Divider />
            </>
          )}
          <Group.ContextMenuItem
            ids={ids}
            shape={shape}
            rootID={rootID}
            onClick={() => group(props)}
          />
        </>
      )}
      {hasDeletePermission && <ContextMenu.DeleteItem onClick={handleDelete} />}
      {(hasUpdatePermission || hasDeletePermission) && <Menu.Divider />}
      {isSingle && (
        <>
          <Export.ContextMenuItem onClick={() => handleExport(first.id.key)} />
          <Link.CopyContextMenuItem
            onClick={() => handleLink({ name: first.name, ontologyID: firstID })}
          />
          <Ontology.CopyPropertiesContextMenuItem {...props} />
          <Menu.Divider />
        </>
      )}
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const handleSelect: Ontology.HandleSelect = ({
  client,
  selection,
  openTab,
  handleError,
}) => {
  const names = strings.naturalLanguageJoin(
    selection.map(({ name }) => name),
    "line plot",
  );
  handleError(async () => {
    const linePlot = await client.lineplots.retrieve({
      key: selection[0].id.key,
    });
    openTab({ variant: "resource", resource: lineplot.ontologyID(linePlot.key) });
  }, `Failed to select ${names}`);
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: "lineplot",
  icon: <Icon.LinePlot />,
  hasChildren: false,
  onSelect: handleSelect,
  haulItems: ({ id }) => [Mosaic.createTabCreateHaulItem(ontology.idToString(id))],
  TreeContextMenu,
};
