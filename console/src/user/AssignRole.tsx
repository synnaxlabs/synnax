// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type access } from "@synnaxlabs/client";
import { Access, Button, Flex, Form, Nav, Synnax } from "@synnaxlabs/pluto";
import { status } from "@synnaxlabs/x";
import { useCallback } from "react";

import { Layout } from "@/layout";
import { Modals } from "@/modals";

export const ASSIGN_ROLE_LAYOUT_TYPE = "user_assign_role";

export interface AssignRoleLayoutArgs extends Access.Role.RetrieveQuery {}

export const ASSIGN_ROLE_LAYOUT: Layout.BaseState = {
  key: ASSIGN_ROLE_LAYOUT_TYPE,
  type: ASSIGN_ROLE_LAYOUT_TYPE,
  icon: "User",
  location: "modal",
  name: "Assign Role",
  window: {
    resizable: false,
    size: { height: 200, width: 500 },
    navTop: true,
  },
};

export const AssignRole: Layout.Renderer = ({ layoutKey, onClose }) => {
  const client = Synnax.use();
  const args = Layout.useSelectArgs<AssignRoleLayoutArgs>(layoutKey);
  const { form, save, variant } = Access.Role.useChangeRoleForm({
    query: args,
    afterSave: useCallback(() => onClose(), [onClose]),
  });
  return (
    <Form.Form<typeof Access.Role.changeRoleFormSchema> {...form}>
      <Flex.Box grow empty>
        <Flex.Box
          className="console-form"
          justify="center"
          style={{ padding: "1rem 3rem" }}
          grow
        >
          <Form.Field<access.role.Key> path="role" label="Role">
            {(props) => <Access.Role.Select {...props} />}
          </Form.Field>
        </Flex.Box>
        <Modals.BottomNavBar>
          <Nav.Bar.End style={{ paddingRight: "2rem" }}>
            <Button.Button
              onClick={() => save()}
              variant="filled"
              disabled={client == null}
              status={status.keepVariants(variant, "loading")}
              tooltip={client == null ? "No Core Connected" : undefined}
              tooltipLocation="bottom"
            >
              Assign
            </Button.Button>
          </Nav.Bar.End>
        </Modals.BottomNavBar>
      </Flex.Box>
    </Form.Form>
  );
};
