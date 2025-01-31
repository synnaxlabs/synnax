// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Align, Button, Form, Nav, Synnax, Text, Triggers } from "@synnaxlabs/pluto";
import { Input } from "@synnaxlabs/pluto/input";
import { useMutation } from "@tanstack/react-query";
import { type ReactElement } from "react";
import { useDispatch } from "react-redux";
import { z } from "zod";

import { Layout } from "@/layout";
import { type SliceState } from "@/layout/slice";
import { useSelectActiveKey } from "@/workspace/selectors";
import { add } from "@/workspace/slice";

export const CREATE_LAYOUT_TYPE = "createWorkspace";

export const CREATE_WINDOW_LAYOUT: Layout.State = {
  key: CREATE_LAYOUT_TYPE,
  type: CREATE_LAYOUT_TYPE,
  windowKey: CREATE_LAYOUT_TYPE,
  name: "Workspace.Create",
  icon: "Workspace",
  location: "modal",
  window: { resizable: false, size: { height: 225, width: 625 }, navTop: true },
};

const formSchema = z.object({
  name: z.string().min(1, { message: "Workspace must have a name" }),
});

const SAVE_TRIGGER: Triggers.Trigger = ["Control", "Enter"];

export const Create = ({ onClose }: Layout.RendererProps): ReactElement => {
  const methods = Form.use({ values: { name: "" }, schema: formSchema });

  const client = Synnax.use();
  const dispatch = useDispatch();
  const active = useSelectActiveKey();

  const { mutate, isPending } = useMutation({
    mutationFn: async () => {
      if (!methods.validate() || client == null) return;
      const { name } = methods.value();
      const ws = await client.workspaces.create({
        name,
        layout: Layout.ZERO_SLICE_STATE,
      });
      dispatch(add(ws));
      if (active != null)
        dispatch(Layout.setWorkspace({ slice: ws.layout as SliceState }));
      onClose();
    },
  });

  return (
    <Align.Space style={{ height: "100%" }}>
      <Align.Space
        className="console-form"
        style={{ padding: "1rem 3rem" }}
        justify="center"
        grow
      >
        <Form.Form {...methods}>
          <Form.Field<string> path="name">
            {(p) => (
              <Input.Text
                placeholder="Workspace Name"
                variant="natural"
                autoFocus
                level="h3"
                {...p}
              />
            )}
          </Form.Field>
        </Form.Form>
      </Align.Space>
      <Layout.BottomNavBar>
        <Nav.Bar.Start size="small">
          <Triggers.Text shade={7} level="small" trigger={SAVE_TRIGGER} />
          <Text.Text shade={7} level="small">
            To Create
          </Text.Text>
        </Nav.Bar.Start>
        <Nav.Bar.End>
          <Button.Button
            type="submit"
            form="create-workspace"
            loading={isPending}
            disabled={isPending || client == null}
            tooltip={client == null ? "No Cluster Connected" : "Save to Cluster"}
            onClick={() => mutate()}
            triggers={[SAVE_TRIGGER]}
          >
            Create
          </Button.Button>
        </Nav.Bar.End>
      </Layout.BottomNavBar>
    </Align.Space>
  );
};
