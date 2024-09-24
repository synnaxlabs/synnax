// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, user } from "@synnaxlabs/client";
import { Align, Form } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { z } from "zod";

import { Layout } from "@/layout";
export const SET_LAYOUT_TYPE = "setPermissions";

interface SetModalProps extends Partial<Layout.State> {
  user: user.User;
}

export const setLayout = ({ user, window, ...rest }: SetModalProps): Layout.State => ({
  key: SET_LAYOUT_TYPE,
  type: SET_LAYOUT_TYPE,
  windowKey: SET_LAYOUT_TYPE,
  icon: "Access",
  location: "modal",
  name: `${user.username}.Permissions`,
  window: {
    resizable: false,
    size: { height: 370, width: 700 },
    navTop: true,
    ...window,
  },
  args: user,
  ...rest,
});

const permissionsZ = z.object({
  schematic: z.boolean(),
  userAndPolicies: z.boolean(),
});

const keysZ = z.object({
  schematic: z.string().optional(),
  userAndPolicies: z
    .object({
      user: z.string(),
      policies: z.string(),
    })
    .optional(),
});

const formSchema = permissionsZ.extend({
  keys: keysZ,
});

export const SetModal = (props: Layout.RendererProps): ReactElement => {
  const { layoutKey } = props;
  const user_ = Layout.useSelectArgs<user.User>(layoutKey);

  const isRootUser = user_.rootUser;

  const methods = Form.useSynced<typeof formSchema>({
    key: [user_.key],
    name: "Permissions",
    values: { schematic: false, userAndPolicies: false, keys: {} },
    queryFn: async ({ client }) => {
      if (client == null) throw new Error("Client is not available");
      const policies = await client.access.policy.retrieveFor(
        user.ontologyID(user_.key),
      );
      const userSpecificPolicies = policies.filter(
        (p) => p.subjects.length === 1 && p.subjects[0].key === user_.key,
      );
      let schematicKey: string | undefined = undefined;
      let userKey: string | undefined = undefined;
      let policiesKey: string | undefined = undefined;
      for (const policy of userSpecificPolicies) {
        if (
          policy.objects.length !== 1 ||
          policy.objects[0].key !== "" ||
          !deep.equal(policy.actions, [access.ALL_ACTION])
        )
          continue;
        const type = policy.objects[0].type;
        const key = policy.key;
        if (type === "schematic") schematicKey = key;
        if (type === "user") userKey = key;
        if (type === "policy") policiesKey = key;
      }
      if (
        (userKey != null && policiesKey == null) ||
        (userKey == null && policiesKey != null)
      )
        throw new Error("User and policies must be set together");
      return {
        schematic: schematicKey != null,
        userAndPolicies: userKey != null,
        keys: {
          schematic: schematicKey,
          userAndPolicies:
            userKey != null
              ? { user: userKey, policies: policiesKey as string }
              : undefined,
        },
      };
    },
    applyChanges: async ({ client, values, path, prev }) => {
      if (path === "") return;
      const policy = path as "schematic" | "userAndPolicies";
      const previouslyActive = prev as boolean;

      if (previouslyActive) {
        if (policy === "schematic")
          await client.access.policy.delete(values.keys.schematic as string);
        else
          await client.access.policy.delete([
            values.keys.userAndPolicies?.user as string,
            values.keys.userAndPolicies?.policies as string,
          ]);
        return;
      }

      if (policy === "schematic") {
        const newPolicy = await client.access.policy.create({
          subjects: {
            type: "user",
            key: user_.key,
          },
          objects: "schematic",
          actions: access.ALL_ACTION,
        });
        values.keys.schematic = newPolicy.key;
        return;
      }

      const newPolicies = await client.access.policy.create([
        {
          subjects: {
            type: "user",
            key: user_.key,
          },
          objects: "user",
          actions: access.ALL_ACTION,
        },
        {
          subjects: {
            type: "user",
            key: user_.key,
          },
          objects: "policy",
          actions: access.ALL_ACTION,
        },
      ]);
      let userKey;
      let policyKey;

      for (const policy of newPolicies) {
        if (policy.objects[0].type === "user") {
          userKey = policy.key;
        } else if (policy.objects[0].type === "policy") {
          policyKey = policy.key;
        }
      }
      if (userKey == null || policyKey == null)
        throw new Error("User and policies must be set together");
      values.keys.userAndPolicies = { user: userKey, policies: policyKey };
    },
  });

  if (isRootUser) return <p>Root user can't set permissions</p>;

  return (
    <Align.Space
      className="console-form"
      justify="center"
      style={{ padding: "3rem" }}
      grow
    >
      <Form.Form {...methods}>
        <Form.SwitchField path={"schematic"} label="Schematic" />
        <Form.SwitchField path={"userAndPolicies"} label="User and Policies" />
      </Form.Form>
    </Align.Space>
  );
};
