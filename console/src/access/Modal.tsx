// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, ontology } from "@synnaxlabs/client";
import {
  Align,
  Divider,
  Input,
  Status,
  Synnax,
  Text,
  useAsyncEffect,
} from "@synnaxlabs/pluto";
import { caseconv } from "@synnaxlabs/x";
import { type HTMLAttributes, ReactElement, useState } from "react";

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
  name: `${initial.name}.Permissions`,
  location: "modal",
  icon: "User",
  window: {
    navTop: true,
    showTitle: true,
  },
  args: initial,
});

const actions: access.Action[] = [
  "all",
  "create",
  "rename",
  "retrieve",
  "delete",
] as const;

// This set of code defines a map of types to appear in the permissions modal and the
// strings that they should be displayed as.
type ListedTypes = Exclude<ontology.ResourceType, "allow_all" | "builtin">;

const allowAll: ontology.ResourceType = "allow_all" as const;

type CompleteTypesMap = {
  [K in ListedTypes]: string;
};

// Define the typesMap with all required keys
const typesObject: CompleteTypesMap = {
  channel: "Channels",
  cluster: "Clusters",
  device: "Devices",
  group: "Groups",
  label: "Labels",
  lineplot: "Line Plots",
  node: "Nodes",
  policy: "Policies",
  rack: "Racks",
  range: "Ranges",
  "range-alias": "Range Aliases",
  schematic: "Schematics",
  task: "Tasks",
  user: "Users",
  workspace: "Workspaces",
};

// Convert the object to a Map
const typesMap = new Map<ListedTypes, string>(
  Object.entries(typesObject) as [ListedTypes, string][],
);

const typesArray = Array.from(typesMap.keys());
const half = Math.ceil(typesArray.length / 2);
const typesArrayOne = typesArray.slice(0, half);
const typesArrayTwo = typesArray.slice(half);

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

  const columnProps = {
    disableAll: allowAllExists || currentUserPolicies,
    handleClick,
    policies,
  };

  return (
    <Align.Space
      direction="y"
      grow
      empty
      size={"large"}
      style={{ width: "100%", padding: "1rem 5rem 3rem" }}
    >
      <Text.Text noWrap level="h2" style={{ margin: "auto" }}>
        {args.name} Permissions
      </Text.Text>
      <Align.Space direction="x" style={{ margin: "auto", padding: "0.5rem 0 1rem 0" }}>
        <Text.Text level="h4" style={{ margin: "auto 0" }}>
          Allow All
        </Text.Text>
        <Input.Switch
          value={allowAllExists}
          disabled={currentUserPolicies}
          onChange={() => handleClick(allowAll, "all")}
        />
      </Align.Space>
      {currentUserPolicies && (
        <Text.Text
          level="p"
          style={{ margin: "0 auto 2rem" }}
          color="var(--pluto-error-z)"
        >
          You cannot change your own permissions
        </Text.Text>
      )}
      <Divider.Divider direction="x" style={{ margin: "0 0 2rem" }} />

      <Align.Space direction="x" style={{ maxHeight: 600, overflowY: "scroll" }}>
        <Column types={typesArrayOne} {...columnProps} />
        <Column types={typesArrayTwo} {...columnProps} />
      </Align.Space>
    </Align.Space>
  );
};

interface ColumnProps extends HTMLAttributes<HTMLDivElement> {
  types: ListedTypes[];
  disableAll: boolean;
  policies: access.Policy[];
  handleClick: (type: ListedTypes, action: access.Action) => void;
}

const Column = ({
  disableAll,
  handleClick,
  policies,
  types,
  ...props
}: ColumnProps): ReactElement => (
  <Align.Space direction="y" style={{ width: "50%" }} {...props}>
    {types.map((type, i) => (
      <Align.Space direction="y" size="medium" key={i} style={{ width: "100%" }}>
        {i > 0 && <Divider.Divider direction="x" />}
        <ActionList
          disableAll={disableAll}
          type={type}
          enabledActions={findCorrectPolicy(policies, type)?.actions ?? []}
          onActionChange={(action) => handleClick(type, action)}
        />
      </Align.Space>
    ))}
  </Align.Space>
);

interface ActionListProps {
  disableAll: boolean;
  type: ListedTypes;
  enabledActions: access.Action[];
  onActionChange: (action: access.Action) => void;
}

const ActionList = ({
  disableAll,
  type,
  enabledActions,
  onActionChange,
}: ActionListProps): ReactElement => (
  <Align.Space direction="y" style={{ width: "fit-content" }}>
    <Text.Text style={{ margin: "0 auto" }} level="h4">
      {typesMap.get(type)}
    </Text.Text>
    {actions.map((action, j) => {
      const value = enabledActions.includes(action);
      const disabled =
        disableAll || (action !== "all" && enabledActions.includes("all"));
      return (
        <Align.Space direction="x" size={"small"} key={j}>
          <Input.Switch
            value={value}
            disabled={disabled}
            onChange={() => onActionChange(action)}
          />
          <Text.Text level="p" style={{ margin: "auto 0" }}>
            {caseconv.capitalize(action)}
          </Text.Text>
        </Align.Space>
      );
    })}
  </Align.Space>
);
