// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger } from "@synnaxlabs/client";
import {
  Divider,
  Form,
  Icon,
  type List,
  Menu as PMenu,
  Ranger,
  Status,
} from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { Menu } from "@/components";
import { Layout } from "@/layout";
import { Link } from "@/link";
import { Modals } from "@/modals";
import { useConfirmDelete } from "@/ontology/hooks";
import {
  createChildRangeMenuItem,
  deleteMenuItem,
  viewDetailsMenuItem,
} from "@/range/ContextMenu";
import { createCreateLayout } from "@/range/Create";
import { OVERVIEW_LAYOUT } from "@/range/overview/layout";
import { useSelectKeys } from "@/range/selectors";
import { add, remove } from "@/range/slice";
import { fromClientRange } from "@/range/translate";

export interface ContextMenuProps extends PMenu.ContextMenuMenuProps {
  getItem: List.GetItem<string, ranger.Range>;
}

export const ContextMenu = ({ keys, getItem }: ContextMenuProps) => {
  const ranges = getItem(keys);
  const isNotEmpty = ranges.length !== 0;
  const isSingle = ranges.length === 1;
  const placeLayout = Layout.usePlacer();
  const favoriteKeys = useSelectKeys();
  const someAreFavorites = ranges.some((r) => favoriteKeys.includes(r.key));
  const someAreNotFavorites = ranges.some((r) => !favoriteKeys.includes(r.key));
  const dispatch = useDispatch();
  const ctx = Form.useContext();
  const rename = Modals.useRename();
  const confirm = useConfirmDelete({
    type: "Range",
    description: "Deleting this range will also delete all child ranges.",
  });
  const { update: del } = Ranger.useDelete();
  const handleAddChildRange = () => {
    placeLayout(createCreateLayout({ parent: ranges[0].key }));
  };
  const handleFavorite = () => {
    dispatch(add({ ranges: fromClientRange(ranges) }));
  };
  const handleUnfavorite = () => {
    dispatch(remove({ keys: ranges.map((r) => r.key) }));
  };
  const handleError = Status.useErrorHandler();
  const handleLink = Cluster.useCopyLinkToClipboard();

  const handleSelect: PMenu.MenuProps["onChange"] = {
    details: () => {
      placeLayout({ ...OVERVIEW_LAYOUT, name: ranges[0].name, key: ranges[0].key });
    },
    rename: () => {
      handleError(async () => {
        const renamed = await rename(
          { initialValue: ranges[0].name },
          { icon: "Range", name: "Range.Rename" },
        );
        if (renamed == null) return;
        ctx.set("name", renamed);
      }, "Failed to rename range");
    },
    delete: () => {
      handleError(async () => {
        const confirmed = await confirm(ranges);
        if (!confirmed) return;
        const keys = ranges.map((r) => r.key);
        dispatch(remove({ keys }));
        dispatch(Layout.remove({ keys }));
        del(keys);
      }, "Failed to delete range");
    },
    addChildRange: handleAddChildRange,
    favorite: handleFavorite,
    unfavorite: handleUnfavorite,
    link: () =>
      handleLink({
        name: ranges[0].name,
        ontologyID: ranger.ontologyID(ranges[0].key),
      }),
  };

  return (
    <PMenu.Menu level="small" gap="small" onChange={handleSelect}>
      {isSingle && (
        <>
          {viewDetailsMenuItem}
          <Menu.RenameItem />
          {createChildRangeMenuItem}
          <Divider.Divider x />
        </>
      )}
      {someAreNotFavorites && (
        <PMenu.Item itemKey="favorite">
          <Icon.StarFilled />
          Add to favorites
        </PMenu.Item>
      )}
      {someAreFavorites && (
        <PMenu.Item itemKey="unfavorite">
          <Icon.StarOutlined />
          Remove from favorites
        </PMenu.Item>
      )}
      {(someAreFavorites || someAreNotFavorites) && <Divider.Divider x />}
      {isNotEmpty && (
        <>
          {deleteMenuItem}
          <Divider.Divider x />
        </>
      )}
      {isSingle && (
        <>
          <Link.CopyMenuItem />
          <Divider.Divider x />
        </>
      )}
      <Menu.ReloadConsoleItem />
    </PMenu.Menu>
  );
};
