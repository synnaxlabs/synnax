// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Flex, Form, Nav, Status } from "@synnaxlabs/pluto";
import { type status, TimeStamp } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { type z } from "zod";

import { Label } from "@/label";
import { Layout } from "@/layout";
import { Modals } from "@/modals";
import { Triggers } from "@/triggers";

export type CreateLayoutArgs = Partial<z.infer<typeof Status.formSchema>>;

export const CREATE_LAYOUT_TYPE = "createStatus";

export const CREATE_LAYOUT: Layout.BaseState<CreateLayoutArgs> = {
  key: CREATE_LAYOUT_TYPE,
  type: CREATE_LAYOUT_TYPE,
  location: "modal",
  name: "Status.Create",
  icon: "Status",
  window: { resizable: false, size: { height: 440, width: 700 }, navTop: true },
};

export const createCreateLayout = (
  initial: CreateLayoutArgs = {},
): Layout.BaseState<CreateLayoutArgs> => ({ ...CREATE_LAYOUT, args: initial });

export const Create = ({ layoutKey, onClose }: Layout.RendererProps): ReactElement => {
  const args = Layout.useSelectArgs<CreateLayoutArgs>(layoutKey);
  const { form, save, variant } = Status.useForm({
    query: { key: args?.key },
    autoSave: false,
    initialValues: {
      ...args,
      key: "",
      message: "",
      time: TimeStamp.now(),
      name: "",
      description: "",
      variant: "success",
      labels: [],
    },
    afterSave: () => onClose(),
  });

  return (
    <Flex.Box grow empty>
      <Flex.Box grow empty style={{ padding: "2rem 3rem" }}>
        <Form.Form<typeof Status.formSchema> {...form}>
          <Form.TextField
            path="name"
            inputProps={{
              autoFocus: true,
              level: "h2",
              variant: "text",
              placeholder: "Name",
            }}
          />
          <Form.Field<status.Variant> path="variant" label="Variant">
            {(props) => <Status.SelectVariant {...props} />}
          </Form.Field>
          <Form.TextField
            path="message"
            label="Message"
            inputProps={{ placeholder: "Message" }}
          />
          <Form.Field<string[]> path="labels" required={false}>
            {({ variant, ...p }) => <Label.SelectMultiple zIndex={100} {...p} />}
          </Form.Field>
        </Form.Form>
      </Flex.Box>
      <Modals.BottomNavBar>
        <Triggers.SaveHelpText action="Save" />
        <Nav.Bar.End>
          <Button.Button
            variant="filled"
            onClick={() => save()}
            tooltipLocation="bottom"
            status={variant}
            trigger={Triggers.SAVE}
          >
            Create
          </Button.Button>
        </Nav.Bar.End>
      </Modals.BottomNavBar>
    </Flex.Box>
  );
};
