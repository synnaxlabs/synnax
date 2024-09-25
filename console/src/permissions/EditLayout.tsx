// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { user } from "@synnaxlabs/client";
import { Align, Divider, Form, Status, Text } from "@synnaxlabs/pluto";
import { Fragment, type ReactElement } from "react";

import { Layout } from "@/layout";
import {
  type ConsolePolicy,
  consolePolicyKeysZ,
  consolePolicyRecord,
  consolePolicySet,
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
    size: { height: 350, width: 700 },
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
    // <Align.Space style={{ paddingTop: "2rem", height: "100%" }} grow empty>
    <Align.Space
      // className="console-form"
      // justify="center"
      direction="y"
      style={{ padding: "1rem" }}
    >
      <Text.Text level="h1" style={{ textAlign: "center", width: "100%" }}>
        Permissions for {user_.username}
      </Text.Text>
      <Form.Form {...methods}>
        <Align.Space direction="y">
          {Array.from(consolePolicySet).map((policy, index) => (
            <Fragment key={index}>
              {index > 0 && <Divider.Divider direction="x" />}
              <Align.Space
                direction="x"
                key={policy}
                grow
                justify="center"
                style={{ paddingLeft: "3rem", paddingRight: "3rem" }}
              >
                <Align.Space direction="x" justify="center" style={{ width: "19.5%" }}>
                  <Form.SwitchField
                    path={policy}
                    label={formItems[policy].label}
                    style={{ width: "fit-content" }}
                  />
                </Align.Space>
                <Divider.Divider direction="y" />
                <Align.Space direction="y" justify="center">
                  <Text.Text level="p">{formItems[policy].description}</Text.Text>
                </Align.Space>
              </Align.Space>
            </Fragment>
          ))}
        </Align.Space>
      </Form.Form>
    </Align.Space>
    // </Align.Space>
  );
};

type FormItems = {
  [P in ConsolePolicy]: {
    label: string;
    description: string;
  };
};

const formItems: FormItems = {
  admin: {
    label: "Admin",
    description:
      "Admin permissions allow the user to manage other users, including registering users and setting permissions for those users.",
  },
  schematic: {
    label: "Edit Schematics",
    description:
      "This permission allow the user to create and edit schematics. If the user does not have this permission, they will still be able to control symbols on the schematic.",
  },
};
