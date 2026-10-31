// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Status } from "@synnaxlabs/pluto";
import { TimeStamp } from "@synnaxlabs/x";
import { useState } from "react";

import { Label } from "@/platform/label";
import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";

export interface CreateModalParams {
  /** When provided, the modal edits the existing status with this key. */
  statusKey?: status.Key;
}

const Create = ({ close, statusKey }: Modals.ContentProps<CreateModalParams, void>) => {
  const [now] = useState(() => TimeStamp.now());
  const { form, save, variant } = Status.useForm({
    query: statusKey == null ? null : { key: statusKey },
    autoSave: false,
    initialValues: {
      key: "",
      message: "",
      time: now,
      name: "",
      description: "",
      variant: "success",
      labels: [],
    },
    afterSave: () => close(),
  });

  return (
    <Modal.Frame>
      <Modal.Header icon={<Icon.Status />}>Status.Create</Modal.Header>
      <Modal.Body>
        <Form.Form<typeof Status.formSchema> {...form}>
          <Form.TextField
            path="name"
            showLabel={false}
            padHelpText={false}
            inputProps={{
              autoFocus: true,
              level: "h2",
              variant: "text",
              placeholder: "Status name",
              "aria-label": "Status name",
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
          <Form.Field<string[]> path="labels">
            {(p) => <Label.SelectMultiple zIndex={100} {...p} />}
          </Form.Field>
        </Form.Form>
      </Modal.Body>
      <Modal.Footer>
        <Nav.Bar.End>
          <Button.Button
            variant="filled"
            onClick={() => save()}
            tooltipLocation="bottom"
            status={variant}
            trigger={Triggers.SAVE}
            triggerIndicator
          >
            Create
          </Button.Button>
        </Nav.Bar.End>
      </Modal.Footer>
    </Modal.Frame>
  );
};

export const useCreateModal = Modals.create<CreateModalParams>(Create);
