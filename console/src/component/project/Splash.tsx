// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/project/Splash.css";

import { project, status, UnexpectedError } from "@synnaxlabs/client";
import { Logo } from "@synnaxlabs/media";
import {
  Access,
  Button,
  Component,
  Flex,
  Form,
  List,
  Nav,
  OS,
  Project as PProject,
  Select,
  Text,
} from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useEffect } from "react";
import { useDispatch } from "react-redux";

import { CSS } from "@/component/css";
import { Nav as ServiceNav } from "@/component/nav";
import { Triggers } from "@/component/triggers";
import { Version } from "@/component/version";
import { Window as ServiceWindow } from "@/component/window";
import { Layout } from "@/layout";
import { select } from "@/session/project/slice";

const listItem = Component.renderProp(
  (props: List.ItemProps<project.Key>): ReactElement | null => {
    const { itemKey } = props;
    const p = List.useItem<project.Key, project.Project>(itemKey);
    if (p == null) return null;
    return (
      <Select.ListItem {...props}>
        <Text.Text>{p.name}</Text.Text>
      </Select.ListItem>
    );
  },
);

const SplashNav = (): ReactElement => {
  const os = OS.use();
  return (
    <ServiceNav.Bar location="top" size="6.5rem" bordered data-tauri-drag-region>
      <Nav.Bar.Start data-tauri-drag-region>
        <ServiceWindow.Controls visibleIfOS="macOS" forceOS={os} />
      </Nav.Bar.Start>
      <Nav.Bar.End data-tauri-drag-region justify="end">
        <Version.Badge />
        <ServiceWindow.Controls visibleIfOS="Windows" forceOS={os} />
      </Nav.Bar.End>
    </ServiceNav.Bar>
  );
};

export const Splash = (): ReactElement => {
  const dispatch = Session.useDispatch();
  const hasRetrievePermission = Access.useRetrieveGranted(project.TYPE_ONTOLOGY_ID);
  const hasCreatePermission = Access.useCreateGranted(project.TYPE_ONTOLOGY_ID);
  const { data, retrieve, getItem, subscribe } = PProject.useList();

  // The list pane only mounts once data is non-empty, so the initial fetch cannot
  // rely on the pane's own onFetchMore.
  useEffect(() => {
    retrieve({});
  }, [retrieve]);

  const handleSelect = useCallback(
    (key: project.Key | null) => {
      if (key == null) return;
      const p = getItem(key);
      if (p == null) throw new UnexpectedError(`Project ${key} not found`);
      dispatch(select(p));
      dispatch(Layout.setProject({ slice: p.layout as Layout.SliceState }));
    },
    [dispatch, getItem],
  );

  const { form, save, variant } = PProject.useForm({
    query: {},
    initialValues: { name: "", layout: Layout.ZERO_SLICE_STATE },
    afterSave: ({ value }) => {
      const { key, name } = value();
      if (key == null) throw new UnexpectedError("Project key is null");
      dispatch(select({ key, name }));
    },
  });

  return (
    <Flex.Box y empty className={CSS.B("project-splash")}>
      <SplashNav />
      <Flex.Box
        y
        align="center"
        justify="center"
        background={1}
        gap="huge"
        grow
        data-tauri-drag-region
        className={CSS.BE("project-splash", "content")}
      >
        <Logo
          variant="title"
          className={CSS.BE("project-splash", "logo")}
          data-tauri-drag-region
        />
        <Flex.Box
          pack
          x
          className={CSS.BE("project-splash", "container")}
          grow={false}
          rounded={1.5}
          background={0}
        >
          {hasRetrievePermission && data.length > 0 && (
            <Flex.Box y className={CSS.BE("project-splash", "list")} bordered empty>
              <Flex.Box align="center" justify="center" style={{ height: "6rem" }}>
                <Text.Text level="h4" color={11} weight={450}>
                  Open a Project
                </Text.Text>
              </Flex.Box>
              <Select.Frame
                data={data}
                onChange={handleSelect}
                getItem={getItem}
                subscribe={subscribe}
                onFetchMore={() => retrieve({})}
              >
                <List.Items grow bordered>
                  {listItem}
                </List.Items>
              </Select.Frame>
            </Flex.Box>
          )}
          <Flex.Box
            y
            align="center"
            justify="center"
            gap="huge"
            className={CSS.BE("project-splash", "form")}
            bordered
            grow
            shrink={false}
          >
            <Text.Text level="h2" color={11} weight={450}>
              New Project
            </Text.Text>
            {hasCreatePermission ? (
              <>
                <Form.Form<typeof PProject.formSchema> {...form}>
                  <Flex.Box y full="x" empty>
                    <Form.TextField
                      path="name"
                      inputProps={{
                        placeholder: "Project name",
                        autoFocus: true,
                        size: "large",
                      }}
                    />
                  </Flex.Box>
                </Form.Form>
                <Button.Button
                  onClick={() => save()}
                  status={status.keepVariants(variant, "loading")}
                  trigger={Triggers.SAVE}
                  variant="filled"
                  size="large"
                >
                  Create Project
                </Button.Button>
              </>
            ) : (
              <Text.Text level="p" color={9}>
                You do not have permission to create a project.
              </Text.Text>
            )}
          </Flex.Box>
        </Flex.Box>
      </Flex.Box>
    </Flex.Box>
  );
};
