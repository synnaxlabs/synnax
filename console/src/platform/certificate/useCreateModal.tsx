// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type certificate, status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Triggers as PTriggers } from "@synnaxlabs/lyra/triggers";
import { Certificate, Synnax } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { z } from "zod";

import { readCertificate, readPrivateKey } from "@/platform/certificate/pem";
import { PEMInput } from "@/platform/certificate/PEMInput";
import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";

const formZ = z.object({
  name: z.string().trim().min(1, "Name is required"),
  pem: z.string().trim().min(1, "Certificate is required"),
  privateKeyPem: z.string(),
});

const path = deep.path<z.infer<typeof formZ>>();

const CERTIFICATE_EXTENSIONS = ["pem", "crt", "cer", "der"];

const KEY_EXTENSIONS = ["pem", "key"];

const NAME_INPUT_PROPS: Partial<Input.TextProps> = {
  level: "h2",
  variant: "text",
  placeholder: "Certificate name",
  "aria-label": "Certificate name",
};

const FOCUSED_NAME_INPUT_PROPS: Partial<Input.TextProps> = {
  ...NAME_INPUT_PROPS,
  autoFocus: true,
};

export interface CreateModalParams {
  /** The name a created certificate starts with. */
  name?: string;
}

const Create = ({
  close,
  name: initialName = "",
}: Modals.ContentProps<CreateModalParams, certificate.Key>): ReactElement => {
  const client = Synnax.use();
  const form = Form.use({
    values: { name: initialName, pem: "", privateKeyPem: "" },
    schema: formZ,
  });
  const { update, variant } = Certificate.useCreate({
    afterSuccess: ({ data }) => close(data.key),
  });
  const busy = variant === "loading";
  Modals.useBusy(busy);
  // A given name moves focus to the certificate.
  const named = initialName.length > 0;
  const handleSave = () => {
    if (!form.validate()) return;
    const { name, pem, privateKeyPem } = form.value();
    update({
      name: name.trim(),
      source: "stored",
      pem,
      privateKeyPem: privateKeyPem.trim() === "" ? undefined : privateKeyPem,
    });
  };
  return (
    <Modal.Frame>
      <Modal.Header icon={<Icon.Certificate />}>Certificate.Create</Modal.Header>
      <Modal.Body>
        <Form.Form<typeof formZ> {...form}>
          <Form.TextField
            path={path.name}
            showLabel={false}
            padHelpText={false}
            inputProps={
              initialName === "" ? FOCUSED_NAME_INPUT_PROPS : NAME_INPUT_PROPS
            }
          />
          <Form.Field<string> path={path.pem} label="Certificate" emptiable>
            {({ value, onChange }) => (
              <PEMInput
                value={value}
                onChange={onChange}
                title="Select certificate"
                extensions={CERTIFICATE_EXTENSIONS}
                read={readCertificate}
                placeholder="-----BEGIN CERTIFICATE-----"
                autoFocus={named}
              />
            )}
          </Form.Field>
          <Form.Field<string> path={path.privateKeyPem} label="Private key" emptiable>
            {({ value, onChange }) => (
              <PEMInput
                value={value}
                onChange={onChange}
                title="Select private key"
                extensions={KEY_EXTENSIONS}
                read={readPrivateKey}
                placeholder="-----BEGIN PRIVATE KEY-----"
              />
            )}
          </Form.Field>
        </Form.Form>
      </Modal.Body>
      <Modal.Footer>
        <Nav.Bar.End x align="center">
          <Button.Button
            variant="outlined"
            onClick={() => close()}
            disabled={busy}
            triggerIndicator={PTriggers.ESCAPE}
          >
            Cancel
          </Button.Button>
          <Button.Button
            variant="filled"
            status={status.keepVariants(variant, "loading")}
            disabled={client == null}
            onClick={handleSave}
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

/**
 * Opens a modal that creates a stored certificate from pasted PEM or a file, with an
 * optional private key. Resolves with the key of the certificate, or null on cancel.
 */
export const useCreateModal = Modals.createPrompt<certificate.Key, CreateModalParams>(
  Create,
);
