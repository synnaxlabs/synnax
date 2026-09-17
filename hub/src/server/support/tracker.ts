// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { LinearClient } from "@linear/sdk";

export const STATUSES = ["open", "done"] as const;
export type Status = (typeof STATUSES)[number];

export interface IssueRef {
  id: string;
  identifier: string;
  url: string;
}

export interface CreateIssueArgs {
  title: string;
  description: string;
  /** customerID attaches a customer request from that Linear customer. */
  customerID?: string;
  need?: string;
}

/**
 * Tracker is the portal's view of Linear: a customer per organization, an issue per
 * thread in the support team, and a comment per message.
 */
export interface Tracker {
  /** ensureCustomer creates the Linear customer for an organization and returns its
   * id. */
  ensureCustomer: (organization: { key: string; name: string }) => Promise<string>;
  createIssue: (args: CreateIssueArgs) => Promise<IssueRef>;
  comment: (issueID: string, body: string) => Promise<void>;
  /** statuses reads the workflow state of each issue, open or done. */
  statuses: (issueIDs: string[]) => Promise<Map<string, Status>>;
  /** reopen moves a done issue back to the team's triage or first open state. */
  reopen: (issueID: string) => Promise<void>;
}

const DONE_TYPES = new Set(["completed", "canceled"]);

/** stateStatus maps a Linear workflow state type to a thread status. */
export const stateStatus = (type: string): Status =>
  DONE_TYPES.has(type) ? "done" : "open";

const STATES_QUERY = `
  query hubIssueStates($ids: [ID!]!) {
    issues(filter: { id: { in: $ids } }, first: 100) {
      nodes { id state { type } }
    }
  }
`;

interface StatesData {
  issues: { nodes: { id: string; state: { type: string } }[] };
}

const REOPEN_ORDER = ["triage", "backlog", "unstarted", "started"];

/** linear connects a Tracker to a workspace through an API key and a team key. */
export const linear = (apiKey: string, teamKey: string): Tracker => {
  const client = new LinearClient({ apiKey });
  let teamID: Promise<string> | undefined;
  const team = (): Promise<string> =>
    (teamID ??= (async () => {
      const { nodes } = await client.teams({ filter: { key: { eq: teamKey } } });
      if (nodes[0] == null) throw new Error(`linear: no team with key ${teamKey}`);
      return nodes[0].id;
    })());

  return {
    ensureCustomer: async ({ key, name }) => {
      const payload = await client.createCustomer({ name, externalIds: [key] });
      const customer = await payload.customer;
      if (customer == null) throw new Error("linear: customer not created");
      return customer.id;
    },
    createIssue: async ({ title, description, customerID, need }) => {
      const payload = await client.createIssue({
        teamId: await team(),
        title,
        description,
      });
      const issue = await payload.issue;
      if (issue == null) throw new Error("linear: issue not created");
      if (customerID != null)
        await client.createCustomerNeed({
          customerId: customerID,
          issueId: issue.id,
          body: need,
        });
      return { id: issue.id, identifier: issue.identifier, url: issue.url };
    },
    comment: async (issueId, body) => {
      await client.createComment({ issueId, body });
    },
    statuses: async (ids) => {
      const out = new Map<string, Status>();
      if (ids.length === 0) return out;
      const { data } = await client.client.rawRequest<StatesData, { ids: string[] }>(
        STATES_QUERY,
        { ids },
      );
      for (const node of data?.issues.nodes ?? [])
        out.set(node.id, stateStatus(node.state.type));
      return out;
    },
    reopen: async (issueID) => {
      const { nodes } = await client.workflowStates({
        filter: { team: { id: { eq: await team() } } },
      });
      const target = REOPEN_ORDER.map((type) =>
        nodes.find((s) => s.type === type),
      ).find((s) => s != null);
      if (target == null) throw new Error("linear: the support team has no open state");
      await client.updateIssue(issueID, { stateId: target.id });
    },
  };
};
