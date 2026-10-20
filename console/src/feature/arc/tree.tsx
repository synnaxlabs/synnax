// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/arc/tree.css";

import { arc, ontology, query, task } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Text } from "@synnaxlabs/lyra/text";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { Tree as PTree } from "@synnaxlabs/lyra/tree";
import { stopPropagation } from "@synnaxlabs/lyra/util";
import { Access, Arc, Task } from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";

import { Arc as PlatformArc } from "@/platform/arc";
import { ContextMenu } from "@/platform/context-menu";
import { Core } from "@/platform/core";
import { CSS } from "@/platform/css";
import { Framer } from "@/platform/framer";
import { Group } from "@/platform/group";
import { Link } from "@/platform/link";
import { Panel } from "@/platform/panel";
import { Tree } from "@/platform/tree";
import { Session } from "@/session";

const useRename = Tree.createUseRename({
  query: Arc.useRename,
  ontologyID: arc.ontologyID,
  convertKey: String,
});

const useDelete = Tree.createUseDelete({
  type: "Arc",
  query: Arc.useDelete,
  convertKey: String,
  beforeUpdate: async ({ data, store }) => {
    store.dispatch(Session.Arc.remove({ keys: array.toArray(data) }));
    return data;
  },
});

const TreeContextMenu: Tree.ContextMenu = (props) => {
  const {
    client,
    selection: { ids, rootID },
    state: { getResource, shape },
  } = props;
  const keys = ids.map((id) => id.key);
  const hasUpdatePermission = Access.useUpdateGranted(ids);
  const hasDeletePermission = Access.useDeleteGranted(ids);
  const canControl = Framer.useCanCommand();
  const groupFromSelection = Group.useCreateFromSelection();
  const captureDeploy = PlatformArc.useCaptureDeploy();
  const { update: runCommand } = Task.useCommand({ afterSuccess: captureDeploy });
  const rename = useRename(props);
  const handleDelete = useDelete(props);
  const handleLink = Core.useCopyLinkToClipboard();
  const firstID = ids[0];
  const first = getResource(firstID);
  const singleResource = ids.length === 1;
  const redeployTaskKeys = !canControl
    ? []
    : keys
        .map((key) => client.arcs.task.getCached(key))
        .filter(
          (tsk): tsk is task.Task =>
            query.isLive(tsk) && tsk != null && task.drifted(tsk),
        )
        .map(({ key }) => key);
  return (
    <ContextMenu.Menu>
      {redeployTaskKeys.length > 0 && (
        <Menu.Item
          className={CSS.BE("arc", "redeploy-item")}
          itemKey="redeploy"
          onClick={() =>
            runCommand(redeployTaskKeys.map((key) => ({ task: key, type: "start" })))
          }
        >
          <Icon.Refresh />
          Redeploy
        </Menu.Item>
      )}
      <Menu.Divider />
      {hasUpdatePermission && singleResource && (
        <ContextMenu.RenameItem onClick={rename} />
      )}
      {hasUpdatePermission && (
        <Group.ContextMenuItem
          ids={ids}
          shape={shape}
          rootID={rootID}
          onClick={() => groupFromSelection(props)}
        />
      )}
      <Menu.Divider />
      {singleResource && (
        <Link.CopyContextMenuItem
          onClick={() => handleLink({ name: first.name, ontologyID: firstID })}
        />
      )}
      <Menu.Divider />
      {hasDeletePermission && <ContextMenu.DeleteItem onClick={handleDelete} />}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

const Content = ({ resource, icon: _, ...rest }: Tree.ContentProps) => {
  const key = resource.id.key;
  const { data } = Arc.useResult({ key });
  const module = data?.type === "module";
  const hasUpdatePermission = Access.useUpdateGranted(resource.id);
  const canControl = Framer.useCanCommand();
  const captureDeploy = PlatformArc.useCaptureDeploy();
  const { running, onStartStop, taskStatus } = Arc.useTaskControls(
    key,
    resource.name,
    { afterSuccess: captureDeploy },
  );
  const drifted = Arc.useDrifted({ arcKey: key });
  const status = module
    ? (data.status ?? Arc.notCompiledYet(resource.name))
    : taskStatus;
  let message = status.message;
  if (!module)
    message = status.variant === "error" ? "Error" : running ? "Running" : "Stopped";
  return (
    <PTree.Item {...rest} className={CSS.B("arc-tree-item")}>
      <Flex.Box
        x
        align="center"
        gap="small"
        className={CSS.BE("arc-tree-item", "leading")}
      >
        <Flex.Box
          x
          align="center"
          gap="small"
          className={CSS.BE("arc-tree-item", "title")}
        >
          <Text.MaybeEditable
            id={List.itemNameID(ontology.idToString(resource.id))}
            value={resource.name}
            onChange
            allowDoubleClick={false}
            className={CSS.BE("arc-tree-item", "name")}
            overflow="fade"
          />
          {data?.mode === "graph" && <PlatformArc.MODE_ICONS.graph />}
        </Flex.Box>
        {!module && drifted && (
          <Tooltip.Dialog>
            <Text.Text level="small">Configuration changed since deploy</Text.Text>
            <Text.Text level="small" status="warning">
              <Icon.Refresh />
            </Text.Text>
          </Tooltip.Dialog>
        )}
      </Flex.Box>
      <Text.Text
        level="small"
        status={status.variant}
        overflow="ellipsis"
        className={CSS.BE("arc-tree-item", "state")}
      >
        {message}
      </Text.Text>
      {module && (
        <Flex.Box center className={CSS.BE("arc-tree-item", "control")}>
          <PlatformArc.TYPE_ICONS.module />
        </Flex.Box>
      )}
      {!module && hasUpdatePermission && canControl && (
        <Button.Button
          className={CSS.BE("arc-tree-item", "control")}
          variant="outlined"
          size="tiny"
          status={status.variant === "loading" ? "loading" : running ? "error" : undefined}
          onClick={onStartStop}
          onDoubleClick={stopPropagation}
          tooltip={`${running ? "Stop" : "Start"} ${resource.name}`}
        >
          {running ? <Icon.Stop /> : <Icon.Play />}
        </Button.Button>
      )}
    </PTree.Item>
  );
};

const TreeItem = Tree.createItem({
  type: "arc",
  icon: <Icon.Arc />,
  hasChildren: false,
  canDrop: () => true,
  useOnSelect: Panel.useOpenResource,
  Content,
  ContextMenu: TreeContextMenu,
});

export const TREE_ITEMS = { arc: TreeItem } satisfies Tree.Items;
