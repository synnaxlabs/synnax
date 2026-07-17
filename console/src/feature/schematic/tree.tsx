// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, ranger, schematic, type Synnax as Client } from "@synnaxlabs/client";
import {
  Access,
  type Flux,
  Icon,
  List,
  Menu,
  Mosaic,
  Schematic as Base,
  Status,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { array, strings } from "@synnaxlabs/x";
import { useCallback } from "react";

import { useExport } from "@/feature/schematic/export";
import { Symbol } from "@/feature/schematic/symbol";
import { Cluster } from "@/platform/cluster";
import { ContextMenu } from "@/platform/context-menu";
import { Export } from "@/platform/export";
import { Group } from "@/platform/group";
import { Layout } from "@/platform/layout";
import { Link } from "@/platform/link";
import { Range } from "@/platform/range";
import { Schematic } from "@/platform/schematic";
import { Tree } from "@/platform/tree";
import { Session } from "@/session";

const useName: Tree.UseName = (id) =>
  Base.useRetrieve({ key: id.key }).data?.name ?? "";

const useDelete = Tree.createUseDelete({
  type: "Schematic",
  query: Base.useDelete,
  convertKey: String,
  useName,
  beforeUpdate: async ({ data, removeLayout, store }) => {
    removeLayout(...data);
    store.dispatch(Session.Schematic.remove({ keys: array.toArray(data) }));
    return data;
  },
});

const useCopy = (props: Tree.ContextMenuProps): (() => void) => {
  const {
    selection: { ids },
  } = props;
  const schematics = Base.useRetrieveMultiple({ keys: ids.map((id) => id.key) }).data;
  const rename = Base.useRename();
  const copy = Base.useCopy({
    afterSuccess: useCallback(
      async ({ data }: Flux.AfterSuccessParams<schematic.Schematic>) => {
        const id = schematic.ontologyID(data.key);
        const [name, renamed] = await Text.asyncEdit(
          List.itemNameID(ontology.idToString(id)),
        );
        if (!renamed) return;
        await rename.updateAsync({ key: data.key, name });
      },
      [rename],
    ),
  });
  return () =>
    ids.map((id) => {
      const current = schematics?.find((s) => s.key === id.key)?.name ?? "";
      copy.update({ key: id.key, name: `${current} (copy)`, snapshot: false });
    });
};

export const useRangeSnapshot = () => {
  const addStatus = Status.useAdder();
  const rng = Session.Range.useSelectState();
  const buildMessage = useCallback(
    ({ schematics }: Base.SnapshotParams) =>
      `${strings.naturalLanguageJoin(
        array.toArray(schematics).map((s) => s.name),
        "schematic",
      )} to ${rng?.name ?? "active range"}`,
    [rng],
  );
  const { update } = Base.useSnapshot({
    afterSuccess: useCallback(
      ({ data }: Flux.AfterSuccessParams<Base.SnapshotParams>) =>
        addStatus({
          variant: "success",
          message: `Successfully snapshotted ${buildMessage(data)}`,
        }),
      [buildMessage, addStatus],
    ),
    afterFailure: ({ status, data }: Flux.AfterFailureParams<Base.SnapshotParams>) =>
      addStatus({ ...status, message: `Failed to snapshot ${buildMessage(data)}` }),
  });
  return (schematics: { key: string; name: string }[]) => {
    if (rng == null)
      return addStatus({
        variant: "error",
        message: "Cannot snapshot schematics without an active range",
      });
    const parentID = ranger.ontologyID(rng.key);
    update({ schematics, parentID });
  };
};

const useRename = Tree.createUseRename({
  query: Base.useRename,
  ontologyID: schematic.ontologyID,
  convertKey: String,
  useName,
  beforeUpdate: async ({ data, rollbacks, store, oldName }) => {
    const { key, name } = data;
    store.dispatch(Session.Layout.rename({ key, name }));
    rollbacks.push(() => store.dispatch(Session.Layout.rename({ key, name: oldName })));
    return { ...data, name };
  },
});

const retrieveProperties = async ({
  client,
  store,
  id,
}: Tree.RetrievePropertiesParams) =>
  await Base.retrieveSingle({
    client,
    store: store as Base.FluxSubStore,
    query: { key: id.key },
  });

const TreeContextMenu: Tree.ContextMenu = (props) => {
  const {
    selection: { ids, rootID },
    state: { shape },
  } = props;
  const activeRange = Session.Range.useSelectState();
  const hasCreatePermission = Access.useCreateGranted(schematic.TYPE_ONTOLOGY_ID);
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const handleDelete = useDelete(props);
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const handleCopy = useCopy(props);
  const snapshot = useRangeSnapshot();
  const handleExport = useExport();
  const handleLink = Cluster.useCopyLinkToClipboard();
  const rename = useRename(props);
  const group = Group.useCreateFromSelection();
  const firstID = ids[0];
  const schematics = Base.useRetrieveMultiple({
    keys: ids.map((id) => id.key),
  }).data;
  const nameOf = (id: ontology.ID) =>
    schematics?.find((s) => s.key === id.key)?.name ?? "";
  const hasNoSnapshots = schematics?.every((s) => s.snapshot === false) ?? false;
  return (
    <ContextMenu.Menu>
      {hasDeletePermission && <ContextMenu.DeleteItem onClick={handleDelete} />}
      {hasUpdatePermission && (
        <>
          <ContextMenu.RenameItem onClick={rename} />
          <Group.ContextMenuItem
            ids={ids}
            shape={shape}
            rootID={rootID}
            onClick={() => group(props)}
          />
          <Menu.Divider />
        </>
      )}
      {hasNoSnapshots && hasCreatePermission && (
        <>
          <Range.SnapshotMenuItem
            range={activeRange}
            onClick={() =>
              snapshot(ids.map((id) => ({ key: id.key, name: nameOf(id) })))
            }
          />
          <Menu.Item itemKey="copy" onClick={handleCopy}>
            <Icon.Copy />
            Copy
          </Menu.Item>
          <Menu.Divider />
        </>
      )}
      <Export.ContextMenuItem onClick={() => handleExport(firstID.key)} />
      <Link.CopyContextMenuItem
        onClick={() => handleLink({ name: nameOf(firstID), ontologyID: firstID })}
      />
      <Tree.CopyPropertiesContextMenuItem
        {...props}
        retrieveProperties={retrieveProperties}
      />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const loadSchematic = async (
  client: Client,
  { key }: ontology.ID,
  placeLayout: Layout.Placer,
) => {
  const schematic = await client.schematics.retrieve({ key });
  placeLayout(Schematic.create({ key: schematic.key, name: schematic.name }));
};

const useOnSelect = (): ((entry: Tree.Entry) => void) => {
  const client = Synnax.use();
  const placeLayout = Layout.usePlacer();
  const handleError = Status.useErrorHandler();
  return useCallback(
    (entry) => {
      if (client == null) return;
      loadSchematic(client, entry.id, placeLayout).catch((e: unknown) =>
        handleError(e, `Failed to select ${entry.name}`),
      );
    },
    [client, placeLayout, handleError],
  );
};

const TreeItem = Tree.createItem({
  type: "schematic",
  icon: <Icon.Schematic />,
  hasChildren: false,
  useName,
  useOnSelect,
  haulItems: ({ id }) => [Mosaic.createTabCreateHaulItem(ontology.idToString(id))],
  ContextMenu: TreeContextMenu,
});

export const TREE_ITEMS = {
  schematic: TreeItem,
  ...Symbol.TREE_ITEMS,
} satisfies Tree.Items;
