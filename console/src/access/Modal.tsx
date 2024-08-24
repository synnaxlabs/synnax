// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, ontology } from "@synnaxlabs/client";
import { Align, Menu, Status, Synnax, Text, useAsyncEffect } from "@synnaxlabs/pluto";
import { caseconv } from "@synnaxlabs/x";
import { Fragment, ReactElement, useState } from "react";

import { Layout } from "@/layout";

export const SET_PERMISSIONS_TYPE = "access-permissions";

type Args = {
  name: string;
  subject: ontology.IDPayload;
};

interface accessLayoutProps extends Partial<Layout.State> {
  initial: Args;
}

export const accessLayout = ({ initial }: accessLayoutProps): Layout.State => ({
  windowKey: SET_PERMISSIONS_TYPE,
  key: SET_PERMISSIONS_TYPE,
  type: SET_PERMISSIONS_TYPE,
  name: "Access.Permissions",
  location: "modal",
  icon: "Access",
  window: {
    navTop: false,
    showTitle: true,
  },
  args: initial,
});

const actions: access.Action[] = [
  "all",
  "create",
  "delete",
  "rename",
  "retrieve",
] as access.Action[];

// allow_all is handled separately
const allowAll: ontology.ResourceType = "allow_all";
const typesMap: Map<ontology.ResourceType, string> = new Map([
  ["label", "Label"],
  ["channel", "Channel"],
  ["group", "Group"],
  ["range", "Range"],
  ["range-alias", "Range Alias"],
  ["user", "User"],
  ["workspace", "Workspace"],
  ["schematic", "Schematic"],
  ["lineplot", "Line Plot"],
  ["rack", "Rack"],
  ["device", "Device"],
  ["task", "Task"],
  ["policy", "Policy"],
]);

const findCorrectPolicy = (
  policies: access.Policy[],
  type: ontology.ResourceType,
): access.Policy | undefined =>
  policies.find((policy) =>
    policy.objects.some((object) => object.type === type && object.key === ""),
  );

// This modal is a somewhat hacky approach to implement user permissions. It assumes the
// only policies that exist are policies based on an ontology type (so no policies that
// refer to a specific subject in the ontology). It also assumes that the subject is a
// specific user. In the future, this feature should be changed to allow for more
// general implementations of access control.
export const UserModal = (props: Layout.RendererProps): ReactElement => {
  const client = Synnax.use();
  const addStatus = Status.useAggregator();
  const [policies, setPolicies] = useState<access.Policy[]>([]);
  const { layoutKey } = props;
  const args = Layout.useSelectArgs<Args>(layoutKey);

  const allowAllPolicy = findCorrectPolicy(policies, allowAll);
  const allowAllExists = allowAllPolicy != null;
  const currentUserPolicies = args.name === client?.props.username;

  useAsyncEffect(async () => {
    try {
      if (client == null) {
        setPolicies([]);
        return;
      }
      const policies = await client.access.retrieveFor(args.subject);
      const newPolicies: access.Policy[] = [];
      typesMap.forEach(async (_, type) => {
        if (findCorrectPolicy(policies, type) == undefined) {
          const newPolicy = await client.access.create({
            subjects: args.subject,
            objects: { type, key: "" },
            actions: [],
          });
          newPolicies.push(newPolicy);
        }
      });
      setPolicies([...policies, ...newPolicies]);
    } catch (e) {
      addStatus({
        variant: "error",
        message: "Failed to retrieve policies from cluster",
        description: (e as Error).message,
      });
    }
  }, [client]);

  const handleClick = async (type: ontology.ResourceType, action: access.Action) => {
    if (client == null) {
      addStatus({
        variant: "error",
        message: "Failed to update policy",
        description: "Client is not available",
      });
      return;
    }
    const policy = findCorrectPolicy(policies, type);
    if (type === allowAll) {
      if (policy != null) {
        // allow all exists
        setPolicies(policies.filter((p) => p.key !== policy.key));
        client.access.delete(policy.key);
        return;
      }
      const newPolicy = await client.access.create({
        subjects: [args.subject],
        objects: [{ type: allowAll, key: "" }],
        actions: [action],
      });
      setPolicies([...policies, newPolicy]);
      return;
    }
    // policy should not be null
    if (policy == null) return;
    const newPolicy: access.Policy = {
      ...policy,
      actions: policy.actions.includes(action)
        ? policy.actions.filter((a) => a !== action)
        : [...policy.actions, action],
    };
    setPolicies([...policies.filter((p) => p.key !== policy.key), newPolicy]);
    client.access.create(newPolicy);
  };

  return (
    <Align.Space
      direction="y"
      grow
      empty
      style={{ width: "100%", padding: "1rem 1rem 1rem 1rem" }}
    >
      <Text.Text noWrap level="h2" style={{ margin: "auto" }}>
        User Permissions: {args.name}
      </Text.Text>
      <div style={{ maxHeight: 500, overflowY: "scroll" }}>
        <Menu.Divider />
        <Align.Space direction="x" style={{ padding: "0.5rem 0 1rem 0" }}>
          <Text.Text level="h4" style={{ width: "20%" }}>
            Allow All
          </Text.Text>
          <input
            type="checkbox"
            readOnly
            checked={allowAllExists}
            disabled={currentUserPolicies}
            style={{ margin: "auto 0 auto 0" }}
            onClick={() => {
              handleClick(allowAll, "all");
            }}
          />
        </Align.Space>
        {Array.from(typesMap.keys()).map((type, i) => (
          <Fragment key={i}>
            <Menu.Divider />
            <Align.Space direction="x" style={{ padding: "0.5rem 0 1rem 0" }}>
              <Text.Text level="h4" style={{ width: "20%" }}>
                {typesMap.get(type)}
              </Text.Text>
              {type === "allow_all" && <Text.Text level="h4">Allow</Text.Text>}
              <Align.Space direction="y">
                {actions.map((action, j) => {
                  const actions = findCorrectPolicy(policies, type)?.actions;
                  const checked = actions?.includes(action) ?? false;
                  const disabled =
                    allowAllExists || (action !== "all" && actions?.includes("all"));
                  return (
                    <Align.Space direction="x" size={"small"} key={j}>
                      <input
                        type="checkbox"
                        readOnly
                        disabled={disabled}
                        checked={checked}
                        style={{ margin: "auto 0 auto 0" }}
                        onClick={() => {
                          handleClick(type, action);
                        }}
                      />
                      <label>{caseconv.capitalize(action)}</label>
                    </Align.Space>
                  );
                })}
              </Align.Space>
            </Align.Space>
          </Fragment>
        ))}
      </div>
    </Align.Space>
  );
};
