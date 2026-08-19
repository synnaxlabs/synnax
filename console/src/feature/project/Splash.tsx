// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/project/Splash.css";

import { project, UnexpectedError } from "@synnaxlabs/client";
import {
  Access,
  Button as PButton,
  Flex,
  Header,
  Icon,
  Input,
  List,
  Menu,
  Project as PProject,
  Select,
  Status,
} from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useEffect, useState } from "react";

import { ContextMenu, listItem } from "@/feature/project/Selector";
import { Shell } from "@/feature/shell";
import { Button } from "@/platform/button";
import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { Project as PlatformProject } from "@/platform/project";
import { Session } from "@/session";

/** Full-window project picker shown when the session has no active project. */
export const Splash = (): ReactElement => {
  const dispatch = Session.useDispatch();
  const logout = Session.useLogout();
  const activeClusterKey = Session.Cluster.useSelectSelectedKey();
  const activeCluster = Session.Cluster.useSelectState(activeClusterKey ?? undefined);
  const hasRetrievePermission = Access.useRetrieveGranted(project.TYPE_ONTOLOGY_ID);
  const hasCreatePermission = Access.useCreateGranted(project.TYPE_ONTOLOGY_ID);
  const openCreate = PlatformProject.useCreateModal();
  const { data, retrieve, getItem, subscribe, answered, status } = PProject.useList();
  const { fetchMore, search } = List.usePager({ retrieve, pageSize: 20 });
  const [searchTerm, setSearchTerm] = useState("");

  // The items pane only mounts once data is non-empty, so the initial fetch
  // cannot rely on its own onFetchMore.
  useEffect(() => {
    fetchMore();
  }, [fetchMore]);

  const handleSearch = useCallback(
    (term: string) => {
      setSearchTerm(term);
      search(term);
    },
    [search],
  );

  const handleSelect = useCallback(
    (key: project.Key | null) => {
      if (key == null) return;
      const p = getItem(key);
      if (p == null) throw new UnexpectedError(`Project ${key} not found`);
      dispatch(Session.Project.select(p.key));
    },
    [dispatch, getItem],
  );

  const contextMenu = useCallback<NonNullable<Menu.ContextMenuProps["menu"]>>(
    (props) => <ContextMenu {...props} getItem={getItem} />,
    [getItem],
  );
  const menuProps = Menu.useContextMenu();
  const getIsAnyModalOpen = Session.Modals.useGetIsAnyOpen();

  return (
    <Shell.Frame className={CSS.B("project-splash")} connection={activeCluster}>
      <Menu.ContextMenu menu={contextMenu} {...menuProps} />
      <Select.Frame
        data={data}
        onChange={handleSelect}
        getItem={getItem}
        subscribe={subscribe}
        onFetchMore={fetchMore}
        enableTriggers={() => !getIsAnyModalOpen()}
      >
        <Flex.Box
          y
          grow
          empty
          className={CSS.cls(CSS.BE("shell", "list"), CSS.BE("project-splash", "list"))}
        >
          <Header.Header gap="small" x>
            <Header.Title level="h4" color={11}>
              <Icon.Project />
              Projects
            </Header.Title>
            <Header.Actions>
              <PButton.Button
                variant="text"
                textColor={9}
                size="medium"
                onClick={logout}
              >
                <Icon.Logout />
                Log out
              </PButton.Button>
            </Header.Actions>
          </Header.Header>
          {hasRetrievePermission && (data.length > 0 || searchTerm !== "") && (
            <Input.Text
              value={searchTerm}
              onChange={handleSearch}
              placeholder="Search projects..."
              startContent={<Icon.Search />}
              autoFocus
              flush
              size="large"
              full="x"
              className={CSS.BE("project-splash", "search")}
            />
          )}
          {hasRetrievePermission && data.length > 0 ? (
            <List.Items
              grow
              className={CSS.cls(
                CSS.BE("shell", "items"),
                CSS.BE("project-splash", "items"),
              )}
              onContextMenu={menuProps.open}
            >
              {listItem}
            </List.Items>
          ) : answered ? (
            <Empty.Action
              grow
              message={searchTerm === "" ? "No projects" : "No matching projects"}
            />
          ) : (
            <Flex.Box grow center>
              <Status.Summary status={status} />
            </Flex.Box>
          )}
          {hasCreatePermission && (
            <Button.CreateListItem size="large" onClick={() => openCreate()}>
              New project
            </Button.CreateListItem>
          )}
        </Flex.Box>
      </Select.Frame>
    </Shell.Frame>
  );
};
