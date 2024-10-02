// Copyright 2024 Synnax Labs, Inc.
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
import { Fragment, type ReactElement, useState } from "react";

import { Layout } from "@/layout";
import {
  type ConsolePolicy,
  consolePolicyKeysZ,
  consolePolicyRecord,
  convertKeysToPermissions,
  convertPoliciesToKeys,
  permissionsZ,
} from "@/permissions/permissions";

export const SET_LAYOUT_TYPE = "setPermissions";

interface EditLayoutProps extends Partial<Layout.State> {
  user: user.User;
}

export const editLayout = ({
  user,
  window,
  ...rest
}: EditLayoutProps): Layout.State => ({
  key: SET_LAYOUT_TYPE,
  type: SET_LAYOUT_TYPE,
  windowKey: SET_LAYOUT_TYPE,
  icon: "Access",
  location: "modal",
  name: `Permissions.${user.username}`,
  window: {
    resizable: false,
    size: { height: 400, width: 700 },
    navTop: true,
    ...window,
  },
  args: user,
  ...rest,
});

const initialPermissions = {
  schematic: false,
  admin: false,
  keys: {},
};

const formSchema = permissionsZ.extend({
  keys: consolePolicyKeysZ,
});

export const EditModal = (props: Layout.RendererProps): ReactElement => {
  const { layoutKey, onClose } = props;
  const user_ = Layout.useSelectArgs<user.User>(layoutKey);
  const addStatus = Status.useAggregator();
  const [isPending, setIsPending] = useState(false);

  const methods = Form.useSynced<typeof formSchema>({
    key: [user_.key],
    name: "Permissions",
    values: { ...initialPermissions, keys: {} },
    queryFn: async ({ client }) => {
      if (client == null) throw new Error("Client is not available");
      const policies = await client.access.policy.retrieveFor(
        user.ontologyID(user_.key),
      );
      const userSpecificPolicies = policies.filter(
        (p) => p.subjects.length === 1 && p.subjects[0].key === user_.key,
      );
      const keys = convertPoliciesToKeys(userSpecificPolicies);
      const permissions = convertKeysToPermissions(keys);
      return {
        ...permissions,
        keys,
      };
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
          subjects: {
            type: user.ONTOLOGY_TYPE,
            key: user_.key,
          },
          ...consolePolicyRecord[policy],
        });
        values.keys[policy] = newPolicy.key;
      } catch (e) {
        if (!(e instanceof Error)) throw e;
        addStatus({
          variant: "error",
          message: `Failed to set ${path}`,
          description: e.message,
        });
      } finally {
        setTimeout(() => setIsPending(false), 100);
      }
    },
  });

  const isRootUser = user_.rootUser;
  if (isRootUser) {
    addStatus({
      variant: "error",
      message: "Root user permissions cannot be modified",
    });
    onClose();
    return <></>;
  }

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
      <Layout.BottomNavBar size="6rem">
        <Nav.Bar.Start align="center" size="large">
          <Status.Text variant={isPending ? "loading" : "success"}>
            {isPending ? "Saving" : "Saved"}
          </Status.Text>
        </Nav.Bar.Start>
      </Layout.BottomNavBar>
    </Align.Space>
  );
};
