// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/project/Selector.css";

import { project, UnexpectedError } from "@synnaxlabs/client";
import {
  Access,
  Button,
  Component,
  CSS as PCSS,
  Dialog,
  type Flux,
  Icon,
  List,
  Menu,
  Project,
  Select,
  Text,
} from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";
import { type MouseEvent, type ReactElement, useCallback, useState } from "react";

import { Avatar } from "@/feature/project/Avatar";
import { ContextMenu as CMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Project as PlatformProject } from "@/platform/project";
import { Tree } from "@/platform/tree";
import { Session } from "@/session";

/* A click inside an active rename would bubble to the row, selecting the
   project and closing the dialog mid-edit. */
const stopClicksWhileEditing = (e: MouseEvent<HTMLDivElement>): void => {
  if ((e.target as HTMLElement).isContentEditable) e.stopPropagation();
};

export const listItem = Component.renderProp(
  (props: List.ItemProps<project.Key>): ReactElement | null => {
    const { itemKey } = props;
    const proj = List.useItem<project.Key, project.Project>(itemKey);
    const { update: rename } = Project.useRename();
    const handleRename = useCallback(
      (name: string) => rename({ key: itemKey, name }),
      [itemKey, rename],
    );
    if (proj == null) return null;
    return (
      <Select.ListItem
        {...props}
        className={Menu.CONTEXT_TARGET}
        data-menu-key={itemKey}
        onClickCapture={stopClicksWhileEditing}
      >
        <Avatar name={proj.name} />
        <Text.Editable
          id={PCSS.B(`project-${itemKey}`)}
          value={proj.name}
          onChange={handleRename}
        />
      </Select.ListItem>
    );
  },
);

export interface ContextMenuProps extends Menu.ContextMenuMenuProps {
  getItem: (key: project.Key) => project.Project | undefined;
}

export const ContextMenu = ({
  keys,
  getItem,
}: ContextMenuProps): ReactElement | null => {
  const ids = project.ontologyID(keys);
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const confirm = Tree.useConfirmDelete({ type: "Project" });
  const { update: del } = Project.useDelete({
    beforeUpdate: useCallback(
      async ({ data }: Flux.BeforeUpdateParams<project.Key | project.Key[]>) => {
        const selected = array
          .toArray(data)
          .map(getItem)
          .filter((p) => p != null);
        if (selected.length === 0) return false;
        if (!(await confirm(selected))) return false;
        return data;
      },
      [confirm, getItem],
    ),
  });
  if (keys.length === 0) return null;
  const [key] = keys;
  return (
    <CMenu.Menu>
      {hasUpdatePermission && keys.length === 1 && (
        <>
          <CMenu.RenameItem onClick={() => Text.edit(PCSS.B(`project-${key}`))} />
          <Menu.Divider />
        </>
      )}
      {hasDeletePermission && (
        <>
          <CMenu.DeleteItem onClick={() => del(keys)} />
          <Menu.Divider />
        </>
      )}
      <CMenu.ReloadConsoleItem />
    </CMenu.Menu>
  );
};

// Rotation switch: "inset" pads the tile inside the button, "fill" makes the
// avatar the button's whole face, "ringed" shrinks it inside a same-hue ring
// with a transparent gap that lets the rail background through.
type TriggerPlacement = "inset" | "fill" | "ringed";
const TRIGGER_PLACEMENT: TriggerPlacement = "ringed";

// getItem is a snapshot read, so the trigger would keep a stale name after a rename.
// useItem subscribes to the list instead.
const TriggerAvatar = ({ itemKey }: { itemKey: project.Key }): ReactElement | null => {
  const proj = List.useItem<project.Key, project.Project>(itemKey);
  if (proj == null) return null;
  return <Avatar name={proj.name} />;
};

export const Selector = (): ReactElement | null => {
  const dispatch = Session.useDispatch();
  const activeKey = Session.Project.useSelectSelected();
  const openCreate = PlatformProject.useCreateModal();
  const [dialogVisible, setDialogVisible] = useState(false);
  const { data, retrieve, getItem, subscribe } = Project.useList();
  const { fetchMore, search } = List.usePager({ retrieve, pageSize: 20 });
  const hasCreatePermission = Access.useCreateGranted(project.TYPE_ONTOLOGY_ID);
  const handleChange = useCallback(
    (key: project.Key | null) => {
      if (key == null) return;
      const proj = getItem(key);
      if (proj == null) throw new UnexpectedError(`Project ${key} not found`);
      dispatch(Session.Project.select(proj.key));
      setDialogVisible(false);
    },
    [dispatch, getItem],
  );
  const handleCreate = useCallback(() => {
    setDialogVisible(false);
    openCreate();
  }, [openCreate]);
  const hasRetrievePermission = Access.useRetrieveGranted(project.TYPE_ONTOLOGY_ID);
  const contextMenu = useCallback<NonNullable<Menu.ContextMenuProps["menu"]>>(
    (props) => <ContextMenu {...props} getItem={getItem} />,
    [getItem],
  );
  const menuProps = Menu.useContextMenu();
  if (!hasRetrievePermission) return null;
  return (
    <Menu.ContextMenu menu={contextMenu} {...menuProps}>
      <Dialog.Frame visible={dialogVisible} onVisibleChange={setDialogVisible}>
        <Select.Frame
          data={data}
          value={activeKey}
          onChange={handleChange}
          getItem={getItem}
          subscribe={subscribe}
          onFetchMore={fetchMore}
        >
          <Dialog.Trigger
            size="large"
            variant="text"
            className={CSS(CSS.B("trigger"), CSS.M(`avatar-${TRIGGER_PLACEMENT}`))}
            hideCaret
          >
            <TriggerAvatar itemKey={activeKey} />
          </Dialog.Trigger>
          <Select.Dialog<project.Key>
            className={CSS.B("project-selector-dialog")}
            resourceName="project"
            onSearch={search}
            onContextMenu={menuProps.open}
            footer={
              hasCreatePermission && (
                <Button.Button
                  variant="text"
                  full="x"
                  justify="start"
                  onClick={handleCreate}
                  className={CSS.BE("project-selector", "create")}
                >
                  <Icon.Add />
                  New Project
                </Button.Button>
              )
            }
          >
            {listItem}
          </Select.Dialog>
        </Select.Frame>
      </Dialog.Frame>
    </Menu.ContextMenu>
  );
};
