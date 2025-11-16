// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/workspace/Guard.css";

import { type workspace } from "@synnaxlabs/client";
import {
  Button,
  Flex,
  Form,
  Header,
  Icon,
  type Input,
  List,
  Nav,
  Select,
  Status,
  Workspace,
} from "@synnaxlabs/pluto";
import {
  type PropsWithChildren,
  type ReactElement,
  type ReactNode,
  useCallback,
  useEffect,
} from "react";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { List as WorkspaceList } from "@/workspace/list";
import { useSelectActiveKey } from "@/workspace/selectors";
import { setActive } from "@/workspace/slice";

const WORKSPACE_NAME_INPUT_PROPS: Partial<Input.TextProps> = {
  autoFocus: true,
  level: "h2",
  variant: "text",
  placeholder: "Workspace Name",
};

const CreateDialog = (): ReactElement => {
  const dispatch = useDispatch();
  const { form, save, variant } = Workspace.useForm({
    query: {},
    initialValues: {
      name: "",
      layout: Layout.ZERO_SLICE_STATE,
    },
    afterSave: ({ value }) => {
      const ws = value();
      const { key, name, layout } = ws;
      if (key == null) return;
      dispatch(setActive({ key, name }));
      dispatch(Layout.setWorkspace({ slice: layout as Layout.SliceState }));
    },
  });
  return (
    <Flex.Box
      className={CSS.B("workspace-create")}
      gap="medium"
      bordered
      rounded={2}
      borderColor={6}
    >
      <Header.Header gap="small" x style={{ padding: "0.666rem" }}>
        <Header.Title level="h5" color={9}>
          <Icon.Workspace />
          Create Your First Workspace
        </Header.Title>
      </Header.Header>
      <Flex.Box grow style={{ padding: "2rem 5rem" }}>
        <Form.Form<typeof Workspace.formSchema> {...form}>
          <Form.TextField path="name" inputProps={WORKSPACE_NAME_INPUT_PROPS} />
        </Form.Form>
      </Flex.Box>
      <Nav.Bar location="bottom" size="7rem" bordered>
        <Nav.Bar.End style={{ paddingRight: "1.5rem" }}>
          <Button.Button variant="filled" onClick={() => save()} status={variant}>
            Create
          </Button.Button>
        </Nav.Bar.End>
      </Nav.Bar>
    </Flex.Box>
  );
};

interface SelectListProps
  extends Pick<
    Select.SingleFrameProps<workspace.Key, workspace.Workspace>,
    "data" | "getItem" | "subscribe" | "value" | "onChange"
  > {}

const SelectList = (props: SelectListProps): ReactElement => (
  <Flex.Box
    className={CSS.B("workspace-select")}
    gap="medium"
    bordered
    rounded={2}
    borderColor={6}
    empty
  >
    <Header.Header gap="small" x style={{ padding: "0.666rem" }}>
      <Header.Title level="h4" color={11}>
        <Icon.Workspace />
        Select a Workspace
      </Header.Title>
    </Header.Header>
    <Select.Frame<workspace.Key, workspace.Workspace> {...props}>
      <List.Items grow style={{ height: 300 }}>
        {WorkspaceList.item}
      </List.Items>
    </Select.Frame>
  </Flex.Box>
);

const WorkspaceSelection = (): ReactElement => {
  const dispatch = useDispatch();
  const {
    status,
    variant,
    getItem,
    subscribe,
    retrieve,
    data: workspaces,
  } = Workspace.useList();
  useEffect(() => {
    retrieve({});
  }, [retrieve]);
  const handleSelect = useCallback(
    (key: string | null) => {
      if (key == null) return;
      const ws = getItem(key);
      if (ws == null) return;
      dispatch(setActive(ws));
      dispatch(
        Layout.setWorkspace({ slice: ws.layout as Layout.SliceState, keepNav: false }),
      );
    },
    [dispatch, getItem],
  );

  if (variant !== "success") return <Status.Summary status={status} />;

  return (
    <Flex.Box full>
      <Layout.Nav.Simple />
      <Flex.Box align="center" justify="center" grow>
        {workspaces.length > 0 ? (
          <SelectList
            data={workspaces}
            getItem={getItem}
            subscribe={subscribe}
            onChange={handleSelect}
          />
        ) : (
          <CreateDialog />
        )}
      </Flex.Box>
    </Flex.Box>
  );
};

export const Guard = ({ children }: PropsWithChildren): ReactNode => {
  const active = useSelectActiveKey();
  if (active != null) return children;
  return <WorkspaceSelection />;
};
