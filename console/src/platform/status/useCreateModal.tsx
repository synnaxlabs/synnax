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

import { Label } from "@/platform/label";
import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";

export interface CreateModalParams {
  /** When provided, the modal edits the existing status with this key. */
  statusKey?: status.Key;
}

export const useCreateModal = Modals.create<CreateModalParams>(
  ({ close, statusKey }) => {
    const { form, save, variant } = Status.useForm({
      query: { key: statusKey },
      autoSave: false,
      initialValues: {
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
              {({ preview: _, ...p }) => <Label.SelectMultiple zIndex={100} {...p} />}
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
