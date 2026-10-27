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
import { type ReactElement } from "react";
import { z } from "zod";

import { MAX_NAME_LENGTH } from "@/server/license/limits";
import { navigate, post } from "@/ui/api";
import { Modal } from "@/ui/modal";

const schema = z.object({
  name: z
    .string()
    .trim()
    .min(1, "Name the organization")
    .max(MAX_NAME_LENGTH, "Too long"),
  email: z.email("Enter the admin's email address"),
});

/** RegisterDialog registers an organization and invites its first admin. Staff only. */
export const RegisterDialog = (): ReactElement => (
  <Modal.Frame
    name="Register an organization"
    icon={<Icon.Role />}
    trigger={
      <Dialog.Trigger variant="filled" hideCaret>
        <Icon.Add />
        Register an organization
      </Dialog.Trigger>
    }
  >
    <Content />
  </Modal.Frame>
);

const Content = (): ReactElement => {
  const methods = Form.use({ values: { name: "", email: "" }, schema });
  return (
    <Modal.Form
      methods={methods}
      submit="Register"
      onSubmit={async (value) => {
        const { key } = await post<{ key: string }>("/api/organizations", value);
        navigate(`/admin/organizations/${key}`);
      }}
    >
      <Form.TextField
        path="name"
        label="Name"
        inputProps={{ autoFocus: true, placeholder: "Acme Rockets" }}
      />
      <Form.Field<string>
        path="email"
        label="Admin email"
        helpText="They get a link to join, then invite the rest of the team."
      >
        {(p) => <Input.Text {...p} type="email" placeholder="name@company.com" />}
      </Form.Field>
    </Modal.Form>
  );
};
