// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type secret, status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Triggers as PTriggers } from "@synnaxlabs/lyra/triggers";
import { Secret, Synnax } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { z } from "zod";

import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";

const formZ = z.object({
  name: z.string().trim().min(1, "Name is required"),
  value: z.string().min(1, "Value is required"),
});

const valueFormZ = formZ.pick({ value: true });

const path = deep.path<z.infer<typeof formZ>>();

const NAME_INPUT_PROPS: Partial<Input.TextProps> = {
  level: "h2",
  variant: "text",
  placeholder: "Secret name",
  "aria-label": "Secret name",
};

const FOCUSED_NAME_INPUT_PROPS: Partial<Input.TextProps> = {
  ...NAME_INPUT_PROPS,
  autoFocus: true,
};

const VALUE_INPUT_PROPS: Partial<Input.TextProps> = {
  type: "password",
  placeholder: "Value",
  autoComplete: "off",
};

const FOCUSED_VALUE_INPUT_PROPS: Partial<Input.TextProps> = {
  ...VALUE_INPUT_PROPS,
  autoFocus: true,
};

const NEW_VALUE_INPUT_PROPS: Partial<Input.TextProps> = {
  ...FOCUSED_VALUE_INPUT_PROPS,
  placeholder: "New value",
};

export interface CreateModalParams {
  /**
   * The key of the stored secret whose value the modal replaces. Unset, the modal
   * creates a secret.
   */
  replace?: secret.Key;
  /** The name a created secret starts with. */
  name?: string;
}

interface FooterProps {
  label: string;
  variant?: status.Variant;
  onCancel: () => void;
  onSave: () => void;
}

const Footer = ({ label, variant, onCancel, onSave }: FooterProps): ReactElement => {
  const client = Synnax.use();
  return (
    <Modal.Footer>
      <Nav.Bar.End x align="center">
        <Button.Button
          variant="outlined"
          onClick={onCancel}
          triggerIndicator={PTriggers.ESCAPE}
        >
          Cancel
        </Button.Button>
        <Button.Button
          variant="filled"
          status={status.keepVariants(variant, "loading")}
          disabled={client == null}
          onClick={onSave}
          trigger={Triggers.SAVE}
          triggerIndicator
        >
          {label}
        </Button.Button>
      </Nav.Bar.End>
    </Modal.Footer>
  );
};

interface BodyProps {
  close: (key?: secret.Key) => void;
}

interface CreateProps extends BodyProps {
  name: string;
}

const Create = ({ close, name: initialName }: CreateProps): ReactElement => {
  const form = Form.use({ values: { name: initialName, value: "" }, schema: formZ });
  const { update, variant } = Secret.useCreate({
    afterSuccess: ({ data }) => close(data.key),
  });
  const handleSave = () => {
    if (!form.validate()) return;
    const { name, value } = form.value();
    update({ name: name.trim(), source: "stored", value });
  };
  // A given name moves focus to the value.
  const named = initialName.length > 0;
  return (
    <Modal.Frame>
      <Modal.Header icon={<Icon.Secret />}>Secret.Create</Modal.Header>
      <Modal.Body>
        <Form.Form<typeof formZ> {...form}>
          <Form.TextField
            path={path.name}
            showLabel={false}
            padHelpText={false}
            inputProps={named ? NAME_INPUT_PROPS : FOCUSED_NAME_INPUT_PROPS}
          />
          <Form.TextField
            path={path.value}
            label="Value"
            inputProps={named ? FOCUSED_VALUE_INPUT_PROPS : VALUE_INPUT_PROPS}
          />
        </Form.Form>
      </Modal.Body>
      <Footer
        label="Create"
        variant={variant}
        onCancel={() => close()}
        onSave={handleSave}
      />
    </Modal.Frame>
  );
};

interface ReplaceProps extends BodyProps {
  secretKey: secret.Key;
}

/** Reads the name live, so the replacement keeps a rename made while it is open. */
const Replace = ({ close, secretKey }: ReplaceProps): ReactElement => {
  const { name } = Secret.use({ key: secretKey });
  const form = Form.use({ values: { value: "" }, schema: valueFormZ });
  const { update, variant } = Secret.useCreate({
    afterSuccess: ({ data }) => close(data.key),
  });
  const handleSave = () => {
    if (!form.validate()) return;
    update({ key: secretKey, name, source: "stored", value: form.value().value });
  };
  return (
    <Modal.Frame>
      <Modal.Header icon={<Icon.Secret />}>{["Replace value", name]}</Modal.Header>
      <Modal.Body>
        <Form.Form<typeof valueFormZ> {...form}>
          <Form.TextField
            path={path.value}
            label="New value"
            inputProps={NEW_VALUE_INPUT_PROPS}
          />
        </Form.Form>
      </Modal.Body>
      <Footer
        label="Replace"
        variant={variant}
        onCancel={() => close()}
        onSave={handleSave}
      />
    </Modal.Frame>
  );
};

const Content = ({
  close,
  replace,
  name = "",
}: Modals.ContentProps<CreateModalParams, secret.Key>): ReactElement =>
  replace == null ? (
    <Create close={close} name={name} />
  ) : (
    <Replace close={close} secretKey={replace} />
  );

/** Opens the modal. Resolves with the key of the saved secret, or null on cancel. */
export const useCreateModal = Modals.createPrompt<secret.Key, CreateModalParams>(
  Content,
);
