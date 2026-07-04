// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type access, status } from "@synnaxlabs/client";
import { Access, Button, Form, Icon, Nav, Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Modals } from "@/layered/service/modals";

export interface AssignRoleModalParams {
  userKey: Access.Role.ChangeRoleFormQuery["key"];
  title?: string;
}

export const useAssignRoleModal = Modals.create<AssignRoleModalParams>(
  ({ title, close, userKey }) => {
    const client = Synnax.use();
    const { form, save, variant } = Access.Role.useChangeRoleForm({
      query: { key: userKey },
      afterSave: useCallback(() => close(), [close]),
    });
    return (
      <Form.Form<typeof Access.Role.changeRoleFormSchema> {...form}>
        <Modals.Frame>
          <Modals.Header icon={<Icon.User />}>{title ?? "Assign Role"}</Modals.Header>
          <Modals.Body>
            <Form.Field<access.role.Key> path="role" label="Role">
              {(props) => <Access.Role.Select {...props} />}
            </Form.Field>
          </Modals.Body>
          <Modals.Footer>
            <Nav.Bar.End>
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
          </Modals.Footer>
        </Modals.Frame>
      </Form.Form>
    );
  },
);
