// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError } from "@synnaxlabs/client";
import { Button, Flex, Form, Input, Nav, Synnax, Workspace } from "@synnaxlabs/pluto";
import { status } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { useDispatch } from "react-redux";

import { Layout } from "@/layout";
import { Modals } from "@/modals";
import { Triggers } from "@/triggers";
import { useSelectActiveKey } from "@/workspace/selectors";
import { setActive } from "@/workspace/slice";

export const CREATE_LAYOUT_TYPE = "createWorkspace";

export const CREATE_LAYOUT: Layout.BaseState = {
  key: CREATE_LAYOUT_TYPE,
  type: CREATE_LAYOUT_TYPE,
  name: "Workspace.Create",
  icon: "Workspace",
  location: "modal",
  window: { resizable: false, size: { height: 225, width: 625 }, navTop: true },
};

export const Create = ({ onClose }: Layout.RendererProps): ReactElement => {
  const client = Synnax.use();
  const dispatch = useDispatch();
  const active = useSelectActiveKey();

  const { form, save, variant } = Workspace.useForm({
    query: {},
    initialValues: {
      name: "",
      layout: Layout.ZERO_SLICE_STATE,
    },
    afterSave: ({ value }) => {
      const ws = value();
      const { key, name, layout } = ws;
      if (key == null) throw new UnexpectedError("Workspace key is null");
      dispatch(setActive({ key, name }));
      if (active != null)
        dispatch(Layout.setWorkspace({ slice: layout as Layout.SliceState }));
      onClose();
    },
  });

  return (
    <Flex.Box style={{ height: "100%" }}>
      <Flex.Box
        className="console-form"
        style={{ padding: "1rem 3rem" }}
        justify="center"
        grow
      >
        <Form.Form<typeof Workspace.formSchema> {...form}>
          <Form.Field<string> path="name">
            {(p) => (
              <Input.Text
                placeholder="Workspace Name"
                variant="text"
                autoFocus
                level="h3"
                {...p}
              />
            )}
          </Form.Field>
        </Form.Form>
      </Flex.Box>
      <Modals.BottomNavBar>
        <Triggers.SaveHelpText action="Create" />
        <Nav.Bar.End>
          <Button.Button
            type="submit"
            variant="filled"
            form="create-workspace"
            status={status.keepVariants(variant, "loading")}
            disabled={client == null}
            tooltip={client == null ? "No Cluster Connected" : "Save to Cluster"}
            onClick={() => save()}
            trigger={Triggers.SAVE}
          >
            Create
          </Button.Button>
        </Nav.Bar.End>
      </Modals.BottomNavBar>
    </Flex.Box>
  );
};
