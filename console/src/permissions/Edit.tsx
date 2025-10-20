// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, DisconnectedError, user } from "@synnaxlabs/client";
import { Flex, Form, Icon, Nav, Status, Text } from "@synnaxlabs/pluto";
import { useEffect, useState } from "react";
import { z } from "zod";

import { Layout } from "@/layout";
import { Modals } from "@/modals";

export const EDIT_LAYOUT_TYPE = "setPermissions";

interface EditLayoutArgs {
  user: user.User;
}

export const createEditLayout = (
  user: user.User,
): Layout.BaseState<EditLayoutArgs> => ({
  key: EDIT_LAYOUT_TYPE,
  type: EDIT_LAYOUT_TYPE,
  icon: "Access",
  location: "modal",
  name: `${user.username}.Roles.Edit`,
  window: { resizable: false, size: { height: 300, width: 700 }, navTop: true },
  args: { user },
});

const formSchema = z.object({
  administrator: z.boolean(),
  adminRoleKey: z.string().uuid().optional(),
});
type FormSchema = typeof formSchema;

export const Edit: Layout.Renderer = ({ layoutKey, onClose }) => {
  const {
    user: { key, rootUser },
  } = Layout.useSelectArgs<EditLayoutArgs>(layoutKey);
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();
  const [isPending, setIsPending] = useState(false);

  const methods = Form.useSynced<FormSchema>({
    key: [key],
    name: "Roles",
    values: { administrator: false },
    queryFn: async ({ client }) => {
      if (client == null) throw new DisconnectedError();

      // Get all roles
      const allRoles = await client.roles.retrieve();
      const adminRole = allRoles.find((r: access.role.Role) => r.name === "Administrator");

      if (adminRole == null) {
        return { administrator: false };
      }

      // Get user to check assigned roles
      const u = await client.users.retrieve({ key });
      const hasAdminRole = u.roles.includes(adminRole.key);

      return {
        administrator: hasAdminRole,
        adminRoleKey: adminRole.key,
      };
    },
    applyChanges: async ({ client, values, path }) => {
      setIsPending(true);
      try {
        if (path !== "administrator") return;
        if (values.adminRoleKey == null) return;

        if (values.administrator) {
          // Assign admin role
          await client.users.assignRoles(key, values.adminRoleKey);
        } else {
          // Unassign admin role
          await client.users.unassignRoles(key, values.adminRoleKey);
        }
      } catch (e) {
        handleError(e, `Failed to update role assignment`);
      } finally {
        setTimeout(() => setIsPending(false), 100);
      }
    },
  });

  useEffect(() => {
    if (rootUser) {
      addStatus({
        message: "Root user roles cannot be modified",
        variant: "error",
      });
      onClose();
    }
  }, [rootUser, onClose, addStatus]);
  if (rootUser) return null;

  return (
    <Flex.Box y grow>
      <Flex.Box y grow style={{ padding: "5rem" }}>
        <Form.Form<typeof formSchema> {...methods}>
          <Flex.Box x align="center" gap={8}>
            <Flex.Box y>
              <Text.Text level="h4" color={10} weight={450}>
                <Icon.Access />
                Administrator
              </Text.Text>
              <Text.Text>
                Administrators have full system access, including the ability to manage
                users, configure hardware, and modify all system resources.
              </Text.Text>
            </Flex.Box>
            <Form.SwitchField
              path="administrator"
              showLabel={false}
              padHelpText={false}
            />
          </Flex.Box>
        </Form.Form>
      </Flex.Box>
      <Modals.BottomNavBar gap={6}>
        <Nav.Bar.Start align="center" gap="large">
          <Status.Summary variant={isPending ? "loading" : "success"}>
            {isPending ? "Saving" : "Saved"}
          </Status.Summary>
        </Nav.Bar.Start>
      </Modals.BottomNavBar>
    </Flex.Box>
  );
};
