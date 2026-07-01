// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { Button, Form, Icon, Nav, Status } from "@synnaxlabs/pluto";
import { TimeStamp } from "@synnaxlabs/x";
import { type z } from "zod";

import { Label } from "@/primitive/label";
import { Modals } from "@/primitive/modals";
import { Triggers } from "@/primitive/triggers";

export type CreateModalParams = Partial<z.infer<typeof Status.formSchema>>;

export const useCreateModal = Modals.create<CreateModalParams>(
  ({ close, ...params }) => {
    const { form, save, variant } = Status.useForm({
      query: { key: params.key },
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
      <Modals.Frame>
        <Modals.Header icon={<Icon.Status />}>Status.Create</Modals.Header>
        <Modals.Body>
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
        </Modals.Body>
        <Modals.Footer>
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
        </Modals.Footer>
      </Modals.Frame>
    );
  },
);
