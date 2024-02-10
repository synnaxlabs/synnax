// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useState, type ReactElement } from "react";

import { zodResolver } from "@hookform/resolvers/zod";
import { Icon } from "@synnaxlabs/media";
import { Align, Button, Header, Input, Nav, Synnax } from "@synnaxlabs/pluto";
import { FormProvider, useForm } from "react-hook-form";
import { useDispatch } from "react-redux";
import { z } from "zod";

import { Layout } from "@/layout";
import { useSelectActiveKey } from "@/workspace/selectors";
import { add } from "@/workspace/slice";

export const createWindowLayout = (
  name: string = "Create Workspace",
): Layout.LayoutState => ({
  key: "createWorkspace",
  type: "createWorkspace",
  windowKey: "createWorkspace",
  name,
  location: "window",
  window: {
    resizable: false,
    size: { height: 225, width: 625 },
    navTop: true,
    transparent: true,
  },
});

const formSchema = z.object({ name: z.string().nonempty() });

type CreateFormProps = z.infer<typeof formSchema>;

export const Create = ({ onClose }: Layout.RendererProps): ReactElement => {
  const methods = useForm({
    defaultValues: {
      name: "",
    },
    resolver: zodResolver(formSchema),
  });
  const [loading, setLoading] = useState(false);

  const client = Synnax.use();
  const dispatch = useDispatch();
  const active = useSelectActiveKey();

  const onSubmit = async ({ name }: CreateFormProps): Promise<void> => {
    if (client == null) return;
    try {
      setLoading(true);
      const ws = await client.workspaces.create({
        name,
        layout: Layout.ZERO_SLICE_STATE,
      });
      dispatch(add({ workspaces: [ws] }));
      if (active != null) dispatch(Layout.setWorkspace({ slice: ws.layout }));
      onClose();
    } finally {
      setLoading(false);
    }
  };

  return (
    <Align.Space style={{ height: "100%" }}>
      <Align.Space
        el="form"
        onSubmit={(e) => {
          e.preventDefault();
          void methods.handleSubmit(onSubmit)(e);
        }}
        style={{ flexGrow: 1, padding: "2rem" }}
        id="create-workspace"
        justify="center"
      >
        <FormProvider {...methods}>
          <Input.HFItem className="console-form" name="name">
            {(p) => (
              <Input.Text
                placeholder="Workspace Name"
                variant="natural"
                autoFocus
                level="h3"
                {...p}
              />
            )}
          </Input.HFItem>
        </FormProvider>
      </Align.Space>
      <Nav.Bar location="bottom" size={48}>
        <Nav.Bar.End style={{ padding: "1rem" }}>
          <Button.Button
            type="submit"
            form="create-workspace"
            loading={loading}
            disabled={loading}
          >
            Save
          </Button.Button>
        </Nav.Bar.End>
      </Nav.Bar>
    </Align.Space>
  );
};
