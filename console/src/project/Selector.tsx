// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/project/Selector.css";

import { project, UnexpectedError } from "@synnaxlabs/client";
import {
  Access,
  Button,
  Component,
  Dialog,
  Flex,
  Icon,
  Input,
  List,
  Project,
  Select,
  Synnax,
  Text,
} from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useState } from "react";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import { useSelectSelected } from "@/layered/session/project/selectors";
import { select } from "@/layered/session/project/slice";
import { Layout } from "@/layout";
import { useCreateModal } from "@/project/Create";

const listItem = Component.renderProp(
  (props: List.ItemProps<project.Key>): ReactElement | null => {
    const { itemKey } = props;
    const proj = List.useItem<project.Key, project.Project>(itemKey);
    if (proj == null) return null;
    return (
      <Select.ListItem {...props}>
        <Text.Text>{proj.name}</Text.Text>
      </Select.ListItem>
    );
  },
);

const DIALOG_STYLE = { minHeight: 200, minWidth: 400 };

export const Selector = (): ReactElement | null => {
  const client = Synnax.use();
  const dispatch = useDispatch();
  const active = useSelectSelected();
  const openCreate = useCreateModal();
  const [dialogVisible, setDialogVisible] = useState(false);
  const { data, retrieve, getItem, subscribe } = Project.useList();
  const [search, setSearch] = useState("");
  const handleChange = useCallback(
    (key: project.Key | null) => {
      if (key == null) return;
      const proj = getItem(key);
      if (proj == null) throw new UnexpectedError(`Project ${key} not found`);
      dispatch(select(proj));
      dispatch(Layout.setProject({ slice: proj.layout as Layout.SliceState }));
      setDialogVisible(false);
    },
    [dispatch, getItem],
  );
  const hasCreatePermission = Access.useCreateGranted(project.TYPE_ONTOLOGY_ID);
  const hasRetrievePermission = Access.useRetrieveGranted(project.TYPE_ONTOLOGY_ID);
  if (!hasRetrievePermission) return null;
  return (
    <Dialog.Frame visible={dialogVisible} onVisibleChange={setDialogVisible}>
      <Select.Frame
        data={data}
        value={active?.key}
        onChange={handleChange}
        getItem={getItem}
        subscribe={subscribe}
        onFetchMore={() => retrieve({})}
      >
        <Dialog.Trigger
          size="medium"
          className={CSS.B("trigger")}
          contrast={2}
          weight={400}
        >
          <Icon.Project key="project" />
          {active.name}
        </Dialog.Trigger>
        <Dialog.Dialog style={DIALOG_STYLE} bordered={client == null} borderColor={6}>
          <Flex.Box pack rounded>
            <Input.Text
              size="large"
              rounded
              placeholder={
                <>
                  <Icon.Search key="search" />
                  Search projects
                </>
              }
              contrast={0}
              value={search}
              onChange={(v) => {
                setSearch(v);
                retrieve((p) => ({ ...p, search: v }));
              }}
              full="x"
              style={{ borderBottomLeftRadius: 0 }}
              borderColor={6}
            />
            {hasCreatePermission && (
              <Button.Button
                size="large"
                variant="outlined"
                onClick={() => {
                  setDialogVisible(false);
                  openCreate();
                }}
                gap="small"
                tooltip="Create a new project"
                tooltipLocation={{ y: "bottom" }}
                borderColor={6}
              >
                <Icon.Add />
                New
              </Button.Button>
            )}
          </Flex.Box>
          <List.Items bordered borderColor={6} grow>
            {listItem}
          </List.Items>
        </Dialog.Dialog>
      </Select.Frame>
    </Dialog.Frame>
  );
};
