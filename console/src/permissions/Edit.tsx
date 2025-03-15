// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { user } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Divider, Form, Nav, Status, Text } from "@synnaxlabs/pluto";
import { useEffect, useState } from "react";

import { NULL_CLIENT_ERROR } from "@/errors";
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
      if (client == null) throw NULL_CLIENT_ERROR;
      const policies = await client.access.policy.retrieveFor(user.ontologyID(key));
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
  if (rootUser) return <></>;

  return (
    <Align.Space direction="y" grow>
      <Align.Space direction="y" grow style={{ padding: "5rem" }}>
        <Form.Form {...methods}>
          <Align.Space direction="y" size="large">
            <Align.Space direction="x" align="center" size={8}>
              <Align.Space direction="y">
                <Text.WithIcon
                  startIcon={<Icon.Access />}
                  level="h4"
                  shade={8}
                  weight={450}
                >
                  Admin
                </Text.WithIcon>
                <Text.Text level="p" shade={7}>
                  Allows the user to manage other users, including registering users and
                  setting permissions for those users.
                </Text.Text>
              </Align.Space>
              <Form.SwitchField path="admin" showLabel={false} padHelpText={false} />
            </Align.Space>
            <Divider.Divider direction="x" />
            <Align.Space direction="y">
              <Text.WithIcon
                startIcon={<Icon.Schematic />}
                level="h4"
                shade={8}
                weight={450}
              >
                Schematics
              </Text.WithIcon>
              <Align.Space
                direction="x"
                size={8}
                align="center"
                style={{ marginLeft: "2rem" }}
              >
                <Align.Space direction="y">
                  <Text.Text level="h5" shade={8}>
                    Edit
                  </Text.Text>
                  <Text.Text level="p" shade={7}>
                    Allow the user to create and edit schematics. If the user does not
                    have this permission, they will still be able to control symbols on
                    the schematic.
                  </Text.Text>
                </Align.Space>
                <Form.SwitchField
                  path="schematic"
                  showLabel={false}
                  padHelpText={false}
                />
              </Align.Space>
            </Align.Space>
          </Align.Space>
        </Form.Form>
      </Align.Space>
      <Modals.BottomNavBar size="6rem">
        <Nav.Bar.Start align="center" size="large">
          <Status.Text variant={isPending ? "loading" : "success"}>
            {isPending ? "Saving" : "Saved"}
          </Status.Text>
        </Nav.Bar.Start>
      </Modals.BottomNavBar>
    </Align.Space>
  );
};
