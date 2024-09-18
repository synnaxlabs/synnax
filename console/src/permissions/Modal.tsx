// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type user } from "@synnaxlabs/client";
import { Align, Divider, Form, Synnax, Text, useAsyncEffect } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { Layout } from "@/layout";
import {
  type ConsolePolicy,
  consolePolicyKeysZ,
  consolePolicyMap,
  consolePolicySet,
  convertKeysToPermissions,
  getConsolePolicyKeys,
  getIsRootUser,
  initialPermissions,
  permissionsZ,
} from "@/permissions/permissions";

export const SET_PERMISSIONS_TYPE = "permissions";

interface ModalProps extends Partial<Layout.State> {
  user: user.User;
}

export const layout = ({ user, ...rest }: ModalProps): Layout.State => ({
  ...rest,
  windowKey: SET_PERMISSIONS_TYPE,
  key: SET_PERMISSIONS_TYPE,
  type: SET_PERMISSIONS_TYPE,
  name: `${user.username}.Permissions`,
  location: "modal",
  icon: "User",
  window: {
    navTop: true,
    showTitle: true,
  },
  args: user,
});

const formSchema = permissionsZ.extend({
  keys: consolePolicyKeysZ,
});

// This modal is a somewhat hacky approach to implement user permissions. It assumes the
// only policies that exist are policies based on an ontology type (so no policies that
// refer to a specific subject in the ontology). It also assumes that the subject is a
// specific user. In the future, this feature should be changed to allow for more
// general implementations of access control.
export const Modal = (props: Layout.RendererProps): ReactElement => {
  const { layoutKey } = props;
  const user = Layout.useSelectArgs<user.User>(layoutKey);
  const client = Synnax.use();
  const [isRootUser, setIsRootUser] = useState(false);

  useAsyncEffect(async () => {
    if (client == null) return;
    setIsRootUser(await getIsRootUser(client, user.key));
  }, [client, user]);

  const methods = Form.useSynced<typeof formSchema>({
    key: [user.key],
    name: "Permissions",
    values: { ...deep.copy(initialPermissions), keys: {} },
    queryFn: async ({ client }) => {
      if (client == null) throw new Error("Client is not available");
      const keys = await getConsolePolicyKeys(client, user.key);
      const permissions = convertKeysToPermissions(keys);
      return { ...permissions, keys };
    },
    applyChanges: async ({ client, values, path, prev }) => {
      if (path === "") return;
      const policy = path as ConsolePolicy;
      const previouslyActive = prev as boolean;

      if (previouslyActive) {
        await client.access.delete([values.keys[policy] as string]);
        values.keys[policy] = undefined;
        return;
      }
      const newPolicy = await client.access.create({
        subjects: {
          type: "user",
          key: user.key,
        },
        ...consolePolicyMap[policy],
      });
      values.keys[policy] = newPolicy.key;
    },
  });

  if (isRootUser)
    return (
      <Align.Space
        className="console-form"
        justify="center"
        style={{ padding: "3rem" }}
        grow
      >
        <Text.Text level="h2">Root User</Text.Text>
        <Text.Text level="p">
          The root user has all permissions and cannot be modified.
        </Text.Text>
      </Align.Space>
    );

  const isCurrentUser = user.key === client?.auth?.user?.key;
  if (isCurrentUser)
    return (
      <Align.Space
        className="console-form"
        justify="center"
        style={{ padding: "3rem" }}
        grow
      >
        <Text.Text level="h2">Current User</Text.Text>
        <Text.Text level="p">You cannot modify your own permissions.</Text.Text>
      </Align.Space>
    );

  return (
    <Align.Space
      className="console-form"
      justify="center"
      style={{ padding: "3rem" }}
      grow
    >
      <Form.Form {...methods}>
        {Array.from(consolePolicySet).map((policy, index) => (
          <>
            {index > 0 && <Divider.Divider direction="x" />}
            <Align.Space direction="x" key={policy}>
              <Form.SwitchField path={policy} label={foo[policy].label} />
              <Text.Text level="p">{foo[policy].description}</Text.Text>
            </Align.Space>
          </>
        ))}
      </Form.Form>
    </Align.Space>
  );
};

type PermissionsModalFormItem = {
  [P in ConsolePolicy]: {
    label: string;
    description: string;
  };
};

const foo: PermissionsModalFormItem = {
  admin: {
    label: "Admin Privileges",
    description:
      "Allows the user to register other users, connects nodes to a cluster, and set permissions on other users.",
  },
  schematic: {
    label: "Schematic Privileges",
    description: "Allows the user to create, edit, and delete schematics.",
  },
};
