// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Access, Button, Flex, Form, Nav, Synnax } from "@synnaxlabs/pluto";
import { status } from "@synnaxlabs/x";

import { type Layout } from "@/layout";
import { Modals } from "@/modals";
import { Triggers } from "@/triggers";

export const ROLE_CREATE_LAYOUT_TYPE = "createRole";

export const ROLE_CREATE_LAYOUT: Layout.BaseState = {
  key: ROLE_CREATE_LAYOUT_TYPE,
  type: ROLE_CREATE_LAYOUT_TYPE,
  icon: "Role",
  location: "modal",
  name: "Role.Create",
  window: {
    resizable: false,
    size: { height: 350, width: 650 },
    navTop: true,
  },
};

export const Create: Layout.Renderer = ({ onClose }) => {
  const client = Synnax.use();
  const { form, save, variant } = Access.Role.useForm({
    query: {},
    afterSave: onClose,
  });

  return (
    <Flex.Box grow empty>
      <Flex.Box
        className="console-form"
        justify="center"
        style={{ padding: "1rem 3rem" }}
        grow
      >
        <Form.Form<typeof Access.Role.formSchema> {...form}>
          <Flex.Box y>
            <Form.TextField
              path="name"
              label="Role Name"
              inputProps={{
                variant: "text",
                level: "h2",
                autoFocus: true,
                placeholder: "Administrator",
                full: "x",
              }}
            />
            <Form.TextField
              path="description"
              label="Description"
              inputProps={{
                placeholder: "Role description",
                full: "x",
              }}
            />
          </Flex.Box>
        </Form.Form>
      </Flex.Box>
      <Modals.BottomNavBar>
        <Triggers.SaveHelpText action="Create" />
        <Nav.Bar.End style={{ paddingRight: "2rem" }}>
          <Button.Button
            onClick={() => save()}
            status={status.keepVariants(variant, "loading")}
            disabled={client == null}
            tooltip={
              client == null
                ? "No Core Connected"
                : `Save to ${client.params.name ?? "Synnax"}`
            }
            tooltipLocation="bottom"
            trigger={Triggers.SAVE}
            variant="filled"
          >
            Create
          </Button.Button>
        </Nav.Bar.End>
      </Modals.BottomNavBar>
    </Flex.Box>
  );
};
