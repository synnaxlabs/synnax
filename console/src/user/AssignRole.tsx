// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type access, status } from "@synnaxlabs/client";
import { Access, Button, Flex, Form, Nav, Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Modals } from "@/layered/service/modals";

export interface AssignRoleModalParams extends Access.Role.RetrieveQuery {
  title?: string;
}

export const useAssignRoleModal = Modals.create<AssignRoleModalParams>(
  ({ params, close }) => {
    const client = Synnax.use();
    const { form, save, variant } = Access.Role.useChangeRoleForm({
      query: params,
      afterSave: useCallback(() => close(), [close]),
    });
    return (
      <Form.Form<typeof Access.Role.changeRoleFormSchema> {...form}>
        <Flex.Box grow empty>
          <Modals.Header name={params.title ?? "Assign Role"} icon="User" />
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
                tooltip={client == null ? "No Core connected" : undefined}
                tooltipLocation="bottom"
              >
                Assign
              </Button.Button>
            </Nav.Bar.End>
          </Modals.BottomNavBar>
        </Flex.Box>
      </Form.Form>
    );
  },
);
