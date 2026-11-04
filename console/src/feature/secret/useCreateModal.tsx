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
import { z } from "zod";

import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";

const formZ = z.object({
  name: z.string().trim().min(1, "Name is required"),
  value: z.string().min(1, "Value is required"),
});

const path = deep.path<z.infer<typeof formZ>>();

const NAME_INPUT_PROPS: Partial<Input.TextProps> = {
  autoFocus: true,
  level: "h2",
  variant: "text",
  placeholder: "Secret name",
  "aria-label": "Secret name",
};

const VALUE_INPUT_PROPS: Partial<Input.TextProps> = {
  type: "password",
  placeholder: "Value",
  autoComplete: "off",
};

const NEW_VALUE_INPUT_PROPS: Partial<Input.TextProps> = {
  ...VALUE_INPUT_PROPS,
  autoFocus: true,
  placeholder: "New value",
};

export interface CreateModalParams {
  /** The secret whose value the modal replaces. Unset, the modal creates a secret. */
  secret?: Pick<secret.Secret, "key" | "name">;
}

const Create = ({
  close,
  secret: existing,
}: Modals.ContentProps<CreateModalParams, void>) => {
  const client = Synnax.use();
  const form = Form.use({
    values: { name: existing?.name ?? "", value: "" },
    schema: formZ,
  });
  const { update, variant } = Secret.useCreate({ afterSuccess: () => close() });
  const handleSave = () => {
    if (!form.validate()) return;
    const { name, value } = form.value();
    update({ key: existing?.key, name: name.trim(), source: "stored", value });
  };
  const replacing = existing != null;
  return (
    <Modal.Frame>
      <Modal.Header icon={<Icon.Secret />}>
        {replacing ? ["Replace value", existing.name] : "Secret.Create"}
      </Modal.Header>
      <Modal.Body>
        <Form.Form<typeof formZ> {...form}>
          {!replacing && (
            <Form.TextField
              path={path.name}
              showLabel={false}
              padHelpText={false}
              inputProps={NAME_INPUT_PROPS}
            />
          )}
          <Form.TextField
            path={path.value}
            label={replacing ? "New value" : "Value"}
            inputProps={replacing ? NEW_VALUE_INPUT_PROPS : VALUE_INPUT_PROPS}
          />
        </Form.Form>
      </Modal.Body>
      <Modal.Footer>
        <Nav.Bar.End x align="center">
          <Button.Button
            variant="outlined"
            onClick={() => close()}
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
            {replacing ? "Replace" : "Create"}
          </Button.Button>
        </Nav.Bar.End>
      </Modal.Footer>
    </Modal.Frame>
  );
};

export const useCreateModal = Modals.create<CreateModalParams>(Create);
