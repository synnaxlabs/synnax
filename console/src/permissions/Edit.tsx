// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, user } from "@synnaxlabs/client";
import { Divider, Flex, Form, Icon, Nav, Status, Text } from "@synnaxlabs/pluto";
import { useEffect, useState } from "react";

import { Layout } from "@/layout";
import { Modals } from "@/modals";
import {
  type ConsolePolicy,
  consolePolicyKeysZ,
  consolePolicyRecord,
  convertKeysToPermissions,
  convertPoliciesToKeys,
  permissionsZ,
} from "@/permissions/permissions";

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
  name: `${user.username}.Permissions.Edit`,
  window: { resizable: false, size: { height: 400, width: 700 }, navTop: true },
  args: { user },
});

const INITIAL_PERMISSIONS = { schematic: false, admin: false, keys: {} };

// eslint-disable-next-line @typescript-eslint/no-unused-vars
const formSchema = permissionsZ.extend({ keys: consolePolicyKeysZ });
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
    name: "Permissions",
    values: { ...INITIAL_PERMISSIONS, keys: {} },
    queryFn: async ({ client }) => {
      if (client == null) throw new DisconnectedError();
      const policies = await client.access.policy.retrieve({
        for: user.ontologyID(key),
      });
      const userSpecificPolicies = policies.filter(
        ({ subjects }) => subjects.length === 1 && subjects[0].key === key,
      );
      const keys = convertPoliciesToKeys(userSpecificPolicies);
      const permissions = convertKeysToPermissions(keys);
      return { ...permissions, keys };
    },
    applyChanges: async ({ client, values, path, prev }) => {
      setIsPending(true);
      try {
        if (path === "") return;
        const policy = path as ConsolePolicy;
        const previouslyActive = prev as boolean;
        if (previouslyActive) {
          const key = values.keys[policy];
          if (key == null) return;
          await client.access.policy.delete(key);
          return;
        }
        const newPolicy = await client.access.policy.create({
          subjects: user.ontologyID(key),
          ...consolePolicyRecord[policy],
        });
        values.keys[policy] = newPolicy.key;
      } catch (e) {
        handleError(e, `Failed to set ${path}`);
      } finally {
        setTimeout(() => setIsPending(false), 100);
      }
    },
  });

  useEffect(() => {
    if (rootUser) {
      addStatus({
        message: "Root user permissions cannot be modified",
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
          <Flex.Box y gap="large">
            <Flex.Box x align="center" gap={8}>
              <Flex.Box y>
                <Text.Text level="h4" color={10} weight={450}>
                  <Icon.Access />
                  Admin
                </Text.Text>
                <Text.Text>
                  Allows the user to manage other users, including registering users and
                  setting permissions for those users.
                </Text.Text>
              </Flex.Box>
              <Form.SwitchField path="admin" showLabel={false} padHelpText={false} />
            </Flex.Box>
            <Divider.Divider x />
            <Flex.Box y>
              <Text.Text level="h4" color={10} weight={450}>
                <Icon.Schematic />
                Schematics
              </Text.Text>
              <Flex.Box x gap={8} align="center" style={{ marginLeft: "2rem" }}>
                <Flex.Box y>
                  <Text.Text level="h5" color={10}>
                    Edit
                  </Text.Text>
                  <Text.Text>
                    Allow the user to create and edit schematics. If the user does not
                    have this permission, they will still be able to control symbols on
                    the schematic.
                  </Text.Text>
                </Flex.Box>
                <Form.SwitchField
                  path="schematic"
                  showLabel={false}
                  padHelpText={false}
                />
              </Flex.Box>
            </Flex.Box>
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
