// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/project/SelectOrCreate.css";

import { project, UnexpectedError } from "@synnaxlabs/client";
import { Logo } from "@synnaxlabs/media";
import {
  Access,
  Button,
  Component,
  Flex,
  Icon,
  Input,
  List,
  Project,
  Select,
  Text,
} from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useState } from "react";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { CREATE_LAYOUT } from "@/project/Create";
import { setActive } from "@/project/slice";

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

export const SelectOrCreate = (): ReactElement => {
  const dispatch = useDispatch();
  const placeLayout = Layout.usePlacer();
  const { data, retrieve, getItem, subscribe } = Project.useList();
  const [search, setSearch] = useState("");
  const hasCreatePermission = Access.useCreateGranted(project.TYPE_ONTOLOGY_ID);
  const handleChange = useCallback(
    (key: project.Key | null) => {
      if (key == null) return;
      const p = getItem(key);
      if (p == null) throw new UnexpectedError(`Project ${key} not found`);
      dispatch(setActive(p));
    },
    [dispatch, getItem],
  );
  return (
    <Flex.Box
      y
      align="center"
      justify="center"
      grow
      gap="huge"
      background={1}
      className={CSS.B("project-select-or-create")}
      data-tauri-drag-region
    >
      <Logo variant="title" data-tauri-drag-region />
      <Flex.Box y align="center" gap="large" className={CSS.BE("project", "container")}>
        <Text.Text level="h2" weight={450} color={11}>
          Select a Project
        </Text.Text>
        <Select.Frame<project.Key, project.Project>
          data={data}
          onChange={handleChange}
          getItem={getItem}
          subscribe={subscribe}
          onFetchMore={() => retrieve({})}
        >
          <Flex.Box pack y rounded background={0} bordered borderColor={6}>
            <Input.Text
              size="large"
              value={search}
              placeholder={
                <>
                  <Icon.Search key="search" />
                  Search projects
                </>
              }
              onChange={(v) => {
                setSearch(v);
                retrieve((p) => ({ ...p, search: v }));
              }}
              full="x"
            />
            <List.Items grow style={{ maxHeight: "40vh", minHeight: 150 }}>
              {listItem}
            </List.Items>
            {hasCreatePermission && (
              <Button.Button
                size="large"
                variant="outlined"
                gap="small"
                onClick={() => placeLayout(CREATE_LAYOUT)}
              >
                <Icon.Add />
                New Project
              </Button.Button>
            )}
          </Flex.Box>
        </Select.Frame>
      </Flex.Box>
    </Flex.Box>
  );
};
