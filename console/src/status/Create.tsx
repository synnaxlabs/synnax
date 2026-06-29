// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { Button, Flex, Form, Nav, Status } from "@synnaxlabs/pluto";
import { TimeStamp } from "@synnaxlabs/x";
import { type z } from "zod";

import { Label } from "@/label";
import { Modals } from "@/layered/service/modals";
import { Triggers } from "@/triggers";

export type CreateModalParams = Partial<z.infer<typeof Status.formSchema>>;

export const useCreateModal = Modals.create<CreateModalParams>(({ params, close }) => {
  const { form, save, variant } = Status.useForm({
    query: { key: params?.key },
    autoSave: false,
    initialValues: {
      ...params,
      key: "",
      message: "",
      time: TimeStamp.now(),
      name: "",
      description: "",
      variant: "success",
      labels: [],
    },
    afterSave: () => close(),
  });

  return (
    <Flex.Box grow empty>
      <Modals.Header name="Status.Create" icon="Status" />
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
});
