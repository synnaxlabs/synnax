// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Dialog, Form, Icon, Input, Text } from "@synnaxlabs/pluto";
import { navigate } from "astro:transitions/client";
import { type ReactElement, useCallback } from "react";
import { z } from "zod";

import { post } from "@/portal/ui/api";
import * as Modal from "@/portal/ui/Modal";
import { useAction } from "@/portal/ui/useAction";

const schema = z.object({
  title: z.string().trim().min(1, "Give the thread a title").max(200),
  message: z.string().trim().min(1, "Write a message"),
});

export interface NewThreadDialogProps {
  organizationKey: string;
  /** organizationName is the team the thread is shared with; absent for a personal
   * account. */
  organizationName?: string;
}

/** NewThreadDialog opens a support thread for an organization. */
export const NewThreadDialog = ({
  organizationKey,
  organizationName,
}: NewThreadDialogProps): ReactElement => (
  <Modal.Frame
    name="New support thread"
    icon={<Icon.Feedback />}
    trigger={
      <Dialog.Trigger variant="filled" hideCaret>
        <Icon.Add />
        New thread
      </Dialog.Trigger>
    }
  >
    <Content organizationKey={organizationKey} organizationName={organizationName} />
  </Modal.Frame>
);

const Content = ({
  organizationKey,
  organizationName,
}: NewThreadDialogProps): ReactElement => {
  const methods = Form.use({ values: { title: "", message: "" }, schema });
  const action = useAction(
    useCallback(async () => {
      if (!methods.validate()) return;
      const { title, message } = methods.value();
      const res = await post<{ thread: string }>("/api/portal/support/threads", {
        org: organizationKey,
        title,
        message,
      });
      await navigate(`/portal/support/${res.thread}`);
    }, [methods, organizationKey]),
  );
  return (
    <Form.Form<typeof schema> {...methods}>
      <Modal.Body gap="medium">
        <Text.Text level="p" color={10}>
          Synnax Labs replies here and by email.
          {organizationName != null &&
            ` The thread is shared with every member of ${organizationName}.`}
        </Text.Text>
        <Form.TextField
          path="title"
          label="Title"
          inputProps={{ autoFocus: true, placeholder: "What do you need help with?" }}
        />
        <Form.Field<string> path="message" label="Message">
          {(p) => (
            <Input.Text
              {...p}
              area
              placeholder="Describe the problem. Include the Core version and what you tried."
              style={{ minHeight: "20rem" }}
            />
          )}
        </Form.Field>
      </Modal.Body>
      <Modal.Footer error={action.error}>
        <Modal.Cancel />
        <Button.Button
          variant="filled"
          onClick={action.run}
          status={action.loading ? "loading" : undefined}
          trigger={["Control", "Enter"]}
          triggerIndicator
        >
          Send
        </Button.Button>
      </Modal.Footer>
    </Form.Form>
  );
};
