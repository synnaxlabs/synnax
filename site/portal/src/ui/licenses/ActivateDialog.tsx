// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Text } from "@synnaxlabs/lyra/text";
import { type CSSProperties, type ReactElement } from "react";
import { z } from "zod";

import { MAX_NAME_LENGTH } from "@/server/license/limits";
import { post, reload, save } from "@/ui/api";
import { Modal } from "@/ui/modal";

export const activateSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, "Name the machine")
    .max(MAX_NAME_LENGTH, "Use a shorter name"),
  fingerprint: z.string().trim().min(1, "Paste the host hashes the Core printed"),
});

export const ZERO_ACTIVATE: z.infer<typeof activateSchema> = {
  name: "",
  fingerprint: "",
};

interface Activated {
  key: string;
  activation: string;
  filename: string;
}

/** activate grants a seat to a machine and downloads its license key. */
export const activate = async (
  licenseKey: string,
  value: z.infer<typeof activateSchema>,
): Promise<void> => {
  const res = await post<Activated>(`/api/licenses/${licenseKey}/activate`, value);
  save(new Blob([res.key], { type: "text/plain" }), res.filename);
};

// Inline, because Lyra's size classes outrank a class on min-height.
const HASHES_STYLE: CSSProperties = { minHeight: "14rem" };

/** ActivateFields renders the instructions and the inputs of the activation form. */
export const ActivateFields = (): ReactElement => (
  <>
    <Text.Text level="p" color={10}>
      Start the Core and copy the host hashes it prints. The Console's activation screen
      shows them too. Give the license key you download to the Core with
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
          style={HASHES_STYLE}
        />
      )}
    </Form.Field>
  </>
);

/** ACTIVATE_LABEL is the content of the button that submits the activation form. */
export const ACTIVATE_LABEL = (
  <>
    <Icon.Download />
    Activate and download license key
  </>
);

export interface ActivateDialogProps {
  licenseKey: string;
  label: string;
}

/**
 * ActivateDialog grants a seat to a machine from its host hashes and downloads the
 * license key that machine needs.
 */
export const ActivateDialog = ({
  licenseKey,
  label,
}: ActivateDialogProps): ReactElement => (
  <Modal.Frame
    name={`${label}.Activate a machine`}
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
  const methods = Form.use({ values: ZERO_ACTIVATE, schema: activateSchema });
  return (
    <Modal.Form
      methods={methods}
      submit={ACTIVATE_LABEL}
      onSubmit={async (value) => {
        await activate(licenseKey, value);
        reload();
      }}
    >
      <ActivateFields />
    </Modal.Form>
  );
};
