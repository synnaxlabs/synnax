// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type access, status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Access, Synnax } from "@synnaxlabs/pluto";

import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";

export interface AssignRoleModalParams {
  /** Key of the user whose role is being assigned. */
  userKey: Access.Role.ChangeRoleFormQuery["key"];
  title?: string | string[];
}

const AssignRole = ({
  title,
  close,
  userKey,
}: Modals.ContentProps<AssignRoleModalParams, void>) => {
  const client = Synnax.use();
  const { form, save, variant } = Access.Role.useChangeRoleForm({
    query: { key: userKey },
    afterSave: () => close(),
  });
  return (
    <Form.Form<typeof Access.Role.changeRoleFormSchema> {...form}>
      <Modal.Frame>
        <Modal.Header icon={<Icon.User />}>{title ?? "Role.Assign"}</Modal.Header>
        <Modal.Body>
          <Form.Field<access.role.Key> path="role" label="Role">
            {(props) => <Access.Role.Select {...props} />}
          </Form.Field>
        </Modal.Body>
        <Modal.Footer>
          <Nav.Bar.End>
            <Button.Button
              onClick={() => save()}
              variant="filled"
              disabled={client == null}
              status={status.keepVariants(variant, "loading")}
              tooltip={client == null ? "No Core connected" : undefined}
              tooltipLocation="bottom"
              trigger={Triggers.SAVE}
              triggerIndicator
            >
              Assign
            </Button.Button>
          </Nav.Bar.End>
        </Modal.Footer>
      </Modal.Frame>
    </Form.Form>
  );
};

export const useAssignRoleModal = Modals.create<AssignRoleModalParams>(AssignRole);
