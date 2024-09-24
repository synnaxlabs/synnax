// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { user } from "@synnaxlabs/client";
import {
  Align,
  Button,
  Divider,
  Form,
  Nav,
  Status,
  Text,
  Triggers,
} from "@synnaxlabs/pluto";
import { Fragment, type ReactElement } from "react";

import { Layout } from "@/layout";
import {
  ConsolePolicy,
  consolePolicyKeysZ,
  consolePolicyRecord,
  consolePolicySet,
  convertKeysToPermissions,
  convertPoliciesToKeys,
  permissionsZ,
} from "@/permissions/permissions";
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
    size: { height: 350, width: 700 },
    navTop: true,
    ...window,
  },
  args: user,
  ...rest,
});

const SAVE_TRIGGER: Triggers.Trigger = ["Control", "Enter"];

const initialPermissions = {
  schematic: false,
  admin: false,
  keys: {},
};

const formSchema = permissionsZ.extend({
  keys: consolePolicyKeysZ,
});

export const SetModal = (props: Layout.RendererProps): ReactElement => {
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
    <Align.Space style={{ paddingTop: "2rem", height: "100%" }} grow empty>
      <Align.Space
        className="console-form"
        justify="center"
        style={{ padding: "3rem" }}
        grow
      >
        <Form.Form {...methods}>
          <Align.Space direction="y">
            <Text.Text level="h2">Permissions for {user_.username}</Text.Text>
            {Array.from(consolePolicySet).map((policy, index) => (
              <Fragment key={index}>
                {index > 0 && <Divider.Divider direction="x" />}
                <Align.Space direction="x" key={policy} justify="center">
                  <Form.SwitchField path={policy} label={foo[policy].label} />
                  <Text.Text level="p">{foo[policy].description}</Text.Text>
                </Align.Space>
              </Fragment>
            ))}
          </Align.Space>
        </Form.Form>
      </Align.Space>
      <Nav.Bar location="bottom" size="7.5rem">
        <Nav.Bar.Start style={{ paddingLeft: "2rem" }}>
          <Triggers.Text level="small" trigger={SAVE_TRIGGER} />
          <Text.Text level="small">To Close</Text.Text>
        </Nav.Bar.Start>
        <Nav.Bar.End style={{ paddingRight: "2rem" }}>
          <Button.Button onClick={onClose} triggers={[SAVE_TRIGGER]}>
            Close
          </Button.Button>
        </Nav.Bar.End>
      </Nav.Bar>
    </Align.Space>
  );
};

type FormItems = {
  [P in ConsolePolicy]: {
    label: string;
    description: string;
  };
};

const foo: FormItems = {
  admin: {
    label: "Admin",
    description:
      "Admin permissions allow the user to manage other users, including registering users and setting permissions for those users.",
  },
  schematic: {
    label: "Schematic",
    description:
      "Schematic permissions allow the user to create and edit schematics. If the user does not have schematic permissions, they will still be able to control the schematic.",
  },
};
