// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/project/Selector.css";

import {
  type project,
  project as projectClient,
  UnexpectedError,
  type workspace,
} from "@synnaxlabs/client";
import {
  Access,
  Button,
  Component,
  Dialog,
  Flex,
  Icon,
  Input,
  List,
  Project as PProject,
  Select,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useState } from "react";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { CREATE_LAYOUT as CREATE_PROJECT_LAYOUT } from "@/project/Create";
import { useSelectActive as useSelectActiveProject } from "@/project/selectors";
import { setActive as setActiveProject } from "@/project/slice";
import { CREATE_LAYOUT as CREATE_WORKSPACE_LAYOUT } from "@/workspace/Create";
import { useSelectActive as useSelectActiveWorkspace } from "@/workspace/selectors";
import { setActive as setActiveWorkspace } from "@/workspace/slice";

const projectItem = Component.renderProp(
  (props: List.ItemProps<project.Key>): ReactElement | null => {
    const p = List.useItem<project.Key, project.Project>(props.itemKey);
    if (p == null) return null;
    return (
      <Select.ListItem {...props} selectOnHover>
        <Text.Text>{p.name}</Text.Text>
      </Select.ListItem>
    );
  },
);

const workspaceItem = Component.renderProp(
  (props: List.ItemProps<workspace.Key>): ReactElement | null => {
    const ws = List.useItem<workspace.Key, workspace.Workspace>(props.itemKey);
    if (ws == null) return null;
    return (
      <Select.ListItem {...props}>
        <Text.Text>{ws.name}</Text.Text>
      </Select.ListItem>
    );
  },
);

const DIALOG_STYLE = { minHeight: 300, minWidth: 600 };

export const Selector = (): ReactElement | null => {
  const client = Synnax.use();
  const dispatch = useDispatch();
  const activeProject = useSelectActiveProject();
  const activeWorkspace = useSelectActiveWorkspace();
  const placeLayout = Layout.usePlacer();
  const [dialogVisible, setDialogVisible] = useState(false);
  const [projectSearch, setProjectSearch] = useState("");

  const {
    data: projectData,
    retrieve: retrieveProjects,
    getItem: getProject,
    subscribe: subscribeProjects,
  } = PProject.useList();

  const [hoveredProjectKey, setHoveredProjectKey] = useState<project.Key | null>(
    activeProject?.key ?? null,
  );

  const {
    data: workspaceData,
    retrieve: retrieveWorkspaces,
    getItem: getWorkspace,
    subscribe: subscribeWorkspaces,
  } = PProject.useListWorkspaces();

  const handleProjectChange = useCallback(
    (key: project.Key | null) => {
      if (key == null) return;
      setHoveredProjectKey(key);
      retrieveWorkspaces({ parent: key });
    },
    [retrieveWorkspaces],
  );

  const handleWorkspaceChange = useCallback(
    (key: workspace.Key | null) => {
      if (key == null || hoveredProjectKey == null) return;
      const p = getProject(hoveredProjectKey);
      const ws = getWorkspace(key);
      if (p == null || ws == null)
        throw new UnexpectedError("Project or workspace not found");
      dispatch(setActiveProject({ key: p.key, name: p.name }));
      dispatch(setActiveWorkspace({ key: ws.key, name: ws.name }));
      dispatch(
        Layout.setWorkspace({
          slice: ws.layout as Layout.SliceState,
          keepNav: false,
        }),
      );
      setDialogVisible(false);
    },
    [dispatch, hoveredProjectKey, getProject, getWorkspace],
  );

  const allowCreate = Access.useCreateGranted(projectClient.TYPE_ONTOLOGY_ID);
  const allowView = Access.useRetrieveGranted(projectClient.TYPE_ONTOLOGY_ID);
  if (!allowView) return null;

  const triggerText =
    activeProject != null && activeWorkspace != null
      ? `${activeProject.name} > ${activeWorkspace.name}`
      : activeProject != null
        ? activeProject.name
        : "No project";

  return (
    <Dialog.Frame visible={dialogVisible} onVisibleChange={setDialogVisible}>
      <Dialog.Trigger
        size="medium"
        className={CSS.B("selector")}
        contrast={2}
        weight={400}
      >
        <Icon.Workspace key="project" />
        {triggerText}
      </Dialog.Trigger>
      <Dialog.Dialog
        style={DIALOG_STYLE}
        bordered={client == null}
        borderColor={6}
        pack
        x
      >
        <Flex.Box y grow className={CSS.BEM("selector", "panel", "left")} empty pack>
          <Flex.Box pack rounded>
            <Input.Text
              size="large"
              rounded
              placeholder={
                <>
                  <Icon.Search key="search" />
                  Projects
                </>
              }
              contrast={0}
              value={projectSearch}
              onChange={(v) => {
                setProjectSearch(v);
                retrieveProjects((p) => ({ ...p, search: v }));
              }}
              full="x"
              style={{ borderBottomLeftRadius: 0, borderBottomRightRadius: 0 }}
              borderColor={6}
            />
            {allowCreate && (
              <Button.Button
                size="large"
                variant="outlined"
                onClick={() => {
                  setDialogVisible(false);
                  placeLayout(CREATE_PROJECT_LAYOUT);
                }}
                gap="small"
                tooltip="Create a new project"
                borderColor={6}
              >
                <Icon.Add />
                New
              </Button.Button>
            )}
          </Flex.Box>
          <Select.Frame
            data={projectData}
            value={hoveredProjectKey ?? undefined}
            onChange={handleProjectChange}
            getItem={getProject}
            subscribe={subscribeProjects}
            onFetchMore={() => retrieveProjects({})}
            allowNone
          >
            <List.Items bordered borderColor={6} grow>
              {projectItem}
            </List.Items>
          </Select.Frame>
        </Flex.Box>
        <Flex.Box y grow className={CSS.BE("selector", "panel")} empty pack>
          <Flex.Box pack rounded>
            <Text.Text
              level="p"
              weight={500}
              style={{ padding: "0.75rem 2rem", borderTop: "var(--pluto-border-l5)" }}
              borderColor={8}
              full="x"
              background={2}
            >
              Workspaces
            </Text.Text>
            {allowCreate && (
              <Button.Button
                size="large"
                variant="outlined"
                onClick={() => {
                  setDialogVisible(false);
                  placeLayout({
                    ...CREATE_WORKSPACE_LAYOUT,
                    args: { projectKey: hoveredProjectKey },
                  });
                }}
                gap="small"
                tooltip="Create a new workspace"
                borderColor={6}
              >
                <Icon.Add />
                New
              </Button.Button>
            )}
          </Flex.Box>
          <Select.Frame
            data={workspaceData}
            value={activeWorkspace?.key}
            onChange={handleWorkspaceChange}
            getItem={getWorkspace}
            subscribe={subscribeWorkspaces}
            onFetchMore={() => {
              if (hoveredProjectKey != null)
                retrieveWorkspaces({ parent: hoveredProjectKey });
            }}
            allowNone
          >
            <List.Items bordered borderColor={6} grow>
              {workspaceItem}
            </List.Items>
          </Select.Frame>
        </Flex.Box>
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};
