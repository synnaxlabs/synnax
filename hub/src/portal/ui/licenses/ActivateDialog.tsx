// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Dialog, Form, Icon, Input, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";
import { z } from "zod";

import { post, reload, save } from "@/portal/ui/api";
import * as Modal from "@/portal/ui/Modal";
import { type Action, useAction } from "@/portal/ui/useAction";
import { MAX_NAME_LENGTH } from "@/server/license/machine";

const schema = z.object({
  name: z
    .string()
    .trim()
    .min(1, "Name the machine")
    .max(MAX_NAME_LENGTH, "Use a shorter name"),
  fingerprint: z.string().trim().min(1, "Paste the host hashes the Core printed"),
});
export type ActivateSchema = typeof schema;

interface Activated {
  token: string;
  activation: string;
  filename: string;
}

interface Activate {
  methods: Form.UseReturn<typeof schema>;
  action: Action;
}

/**
 * useActivate builds the form and the action that grant a seat from pasted host
 * hashes and download the machine's token. `onDone` runs after the download.
 */
export const useActivate = (
  licenseKey: string,
  onDone: () => Promise<void>,
): Activate => {
  const methods = Form.use({ values: { name: "", fingerprint: "" }, schema });
  const action = useAction(
    useCallback(async () => {
      if (!methods.validate()) return;
      const { name, fingerprint } = methods.value();
      const res = await post<Activated>(`/api/portal/licenses/${licenseKey}/activate`, {
        name,
        fingerprint,
      });
      save(new Blob([res.token], { type: "text/plain" }), res.filename);
      await onDone();
    }, [methods, licenseKey, onDone]),
  );
  return { methods, action };
};

/** ActivateFields renders the instructions and the inputs of the activation form. */
export const ActivateFields = (): ReactElement => (
  <>
    <Text.Text level="p" color={10}>
      Start the Core and copy the host hashes it prints. The Console's activation screen
      shows them too. Give the token you download to the Core with
      <code>--license-file</code> or through the Console.
    </Text.Text>
    <Form.TextField
      path="name"
      label="Machine name"
      inputProps={{ autoFocus: true, placeholder: "Test stand, site B" }}
    />
    <Form.Field<string> path="fingerprint" label="Host hashes">
      {(p) => (
        <Input.Text
          {...p}
          area
          spellCheck={false}
          placeholder="One hash per line, or separated by commas"
          style={{ minHeight: "14rem" }}
        />
      )}
    </Form.Field>
  </>
);

export const ActivateButton = ({ action }: { action: Action }): ReactElement => (
  <Button.Button
    variant="filled"
    onClick={action.run}
    status={action.loading ? "loading" : undefined}
    trigger={["Control", "Enter"]}
    triggerIndicator
  >
    <Icon.Download />
    Activate and download token
  </Button.Button>
);

export interface ActivateDialogProps {
  licenseKey: string;
  label: string;
}

/**
 * ActivateDialog grants a seat to a machine from its host hashes and downloads the
 * token that machine needs.
 */
export const ActivateDialog = ({
  licenseKey,
  label,
}: ActivateDialogProps): ReactElement => (
  <Modal.Frame
    name={`${label || "License"}.Activate a machine`}
    icon={<Icon.Add />}
    trigger={
      <Dialog.Trigger variant="filled" hideCaret>
        <Icon.Add />
        Activate a machine
      </Dialog.Trigger>
    }
  >
    <Content licenseKey={licenseKey} />
  </Modal.Frame>
);

const Content = ({ licenseKey }: { licenseKey: string }): ReactElement => {
  const { close } = Dialog.useContext();
  const { methods, action } = useActivate(
    licenseKey,
    useCallback(async () => {
      close();
      await reload();
    }, [close]),
  );
  return (
    <Form.Form<ActivateSchema> {...methods}>
      <Modal.Body gap="medium">
        <ActivateFields />
      </Modal.Body>
      <Modal.Footer error={action.error}>
        <Modal.Cancel />
        <ActivateButton action={action} />
      </Modal.Footer>
    </Form.Form>
  );
};
