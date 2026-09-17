// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  PlainClient,
  type PlainSDKError,
  SortDirection,
  type ThreadPartsFragment,
  ThreadsSortField,
  ThreadStatus,
} from "@team-plain/typescript-sdk";
import { z } from "zod";

import { entryNodeZ, type Message, toMessages } from "@/server/support/timeline";

export const STATUSES = ["todo", "snoozed", "done"] as const;
export type Status = (typeof STATUSES)[number];

/** Thread is what the support pages show of a Plain thread. */
export interface Thread {
  id: string;
  ref: string;
  title: string;
  status: Status;
  preview: string | null;
  createdAt: Date;
  changedAt: Date;
}

export interface ThreadDetail extends Thread {
  /** tenantExternalID is the organization key the thread belongs to, or null. */
  tenantExternalID: string | null;
  customerID: string;
  messages: Message[];
}

/** Member is a signed-in user as Plain knows them: a customer keyed by user id. */
export interface Member {
  userID: string;
  email: string;
  name: string;
}

export interface CreateThreadArgs {
  customerID: string;
  organizationKey: string;
  title: string;
  text: string;
}

export interface ReplyArgs {
  customerID: string;
  threadID: string;
  text: string;
}

export interface FeedbackArgs {
  /** email is empty for an anonymous submission. */
  email: string;
  name: string;
  text: string;
  /** page is the site path the feedback was sent from. */
  page: string;
  organizationKey?: string;
}

/**
 * Support is the portal's view of Plain. Organizations are tenants keyed by
 * organization key, members are customers keyed by user id, and every customer
 * message is a chat on a CHAT-channel thread.
 */
export interface Support {
  /** ensureTenant mirrors an organization and returns its Plain tenant id. */
  ensureTenant: (organizationKey: string, name: string) => Promise<string>;
  /**
   * ensureMember mirrors a user into the organization's tenant and returns their
   * Plain customer id.
   */
  ensureMember: (member: Member, organizationKey: string) => Promise<string>;
  listThreads: (organizationKey: string) => Promise<Thread[]>;
  retrieveThread: (threadID: string) => Promise<ThreadDetail | null>;
  createThread: (args: CreateThreadArgs) => Promise<Thread>;
  reply: (args: ReplyArgs) => Promise<void>;
  /** submitFeedback opens a thread from the docs feedback form. */
  submitFeedback: (args: FeedbackArgs) => Promise<Thread>;
}

/** ANONYMOUS is the customer feedback lands on when no email is given. */
export const ANONYMOUS = {
  email: "anonymous@synnaxlabs.com",
  name: "Anonymous visitor",
};

const PAGE = 50;

type Channel = NonNullable<Parameters<PlainClient["createThread"]>[0]["channel"]>;
/** CHAT is the only channel that accepts customer chats. The SDK hides the enum. */
const CHAT = "CHAT" as Channel;

const unwrap = <T>(result: { data?: T; error?: PlainSDKError }): T => {
  if (result.error != null) throw new Error(`plain: ${result.error.message}`);
  return result.data as T;
};

const STATUS: Record<ThreadStatus, Status> = {
  [ThreadStatus.Todo]: "todo",
  [ThreadStatus.Snoozed]: "snoozed",
  [ThreadStatus.Done]: "done",
};

const toThread = (t: ThreadPartsFragment): Thread => ({
  id: t.id,
  ref: t.ref,
  title: t.title,
  status: STATUS[t.status],
  preview: t.previewText,
  createdAt: new Date(t.createdAt.iso8601),
  changedAt: new Date(t.statusChangedAt.iso8601),
});

const THREAD_QUERY = `
  query supportThread($threadId: ID!) {
    thread(threadId: $threadId) {
      id
      ref
      title
      status
      previewText
      createdAt { iso8601 }
      statusChangedAt { iso8601 }
      tenant { externalId }
      customer { id }
      timelineEntries(first: 200, filters: { entryTypes: [CHAT, EMAIL, CUSTOM] }) {
        edges {
          node {
            id
            timestamp { iso8601 }
            actor {
              __typename
              ... on UserActor { user { publicName } }
              ... on CustomerActor { customer { fullName } }
            }
            entry {
              __typename
              ... on ChatEntry { text }
              ... on EmailEntry { textContent markdownContent }
              ... on CustomEntry {
                title
                components {
                  __typename
                  ... on ComponentText { text }
                  ... on ComponentPlainText { plainText }
                }
              }
            }
          }
        }
      }
    }
  }
`;

const threadQueryZ = z.object({
  thread: z
    .object({
      id: z.string(),
      ref: z.string(),
      title: z.string(),
      status: z.enum(ThreadStatus),
      previewText: z.string().nullable(),
      createdAt: z.object({ iso8601: z.string() }),
      statusChangedAt: z.object({ iso8601: z.string() }),
      tenant: z.object({ externalId: z.string() }).nullable(),
      customer: z.object({ id: z.string() }),
      timelineEntries: z.object({ edges: z.array(z.object({ node: entryNodeZ })) }),
    })
    .nullable(),
});

/** plain connects Support to a Plain workspace with a machine user API key. */
export const plain = (apiKey: string): Support => {
  const client = new PlainClient({ apiKey });
  const tenantOf = (organizationKey: string) => ({ externalId: organizationKey });

  const ensureTenant: Support["ensureTenant"] = async (organizationKey, name) => {
    const tenant = unwrap(
      await client.upsertTenant({
        identifier: tenantOf(organizationKey),
        externalId: organizationKey,
        name,
      }),
    );
    return tenant.id;
  };

  const ensureMember: Support["ensureMember"] = async (
    { userID, email, name },
    organizationKey,
  ) => {
    const { customer } = unwrap(
      await client.upsertCustomer({
        identifier: { externalId: userID },
        onCreate: {
          externalId: userID,
          fullName: name,
          email: { email, isVerified: true },
          tenantIdentifiers: [tenantOf(organizationKey)],
        },
        onUpdate: {
          fullName: { value: name },
          email: { email, isVerified: true },
        },
      }),
    );
    unwrap(
      await client.addCustomerToTenants({
        customerIdentifier: { customerId: customer.id },
        tenantIdentifiers: [tenantOf(organizationKey)],
      }),
    );
    return customer.id;
  };

  const listThreads: Support["listThreads"] = async (organizationKey) => {
    const { threads } = unwrap(
      await client.getThreads({
        filters: { tenantIdentifiers: [tenantOf(organizationKey)] },
        sortBy: { field: ThreadsSortField.CreatedAt, direction: SortDirection.Desc },
        first: PAGE,
      }),
    );
    return threads.map(toThread);
  };

  const retrieveThread: Support["retrieveThread"] = async (threadID) => {
    const raw = unwrap(
      await client.rawRequest({
        query: THREAD_QUERY,
        variables: { threadId: threadID },
      }),
    );
    const { thread } = threadQueryZ.parse(raw);
    if (thread == null) return null;
    return {
      id: thread.id,
      ref: thread.ref,
      title: thread.title,
      status: STATUS[thread.status],
      preview: thread.previewText,
      createdAt: new Date(thread.createdAt.iso8601),
      changedAt: new Date(thread.statusChangedAt.iso8601),
      tenantExternalID: thread.tenant?.externalId ?? null,
      customerID: thread.customer.id,
      messages: toMessages(thread.timelineEntries.edges.map((e) => e.node)),
    };
  };

  const reply: Support["reply"] = async ({ customerID, threadID, text }) => {
    unwrap(
      await client.sendCustomerChat({
        customerId: customerID,
        threadId: threadID,
        text,
      }),
    );
  };

  const createThread: Support["createThread"] = async ({
    customerID,
    organizationKey,
    title,
    text,
  }) => {
    const created = unwrap(
      await client.createThread({
        channel: CHAT,
        customerIdentifier: { customerId: customerID },
        tenantIdentifier: tenantOf(organizationKey),
        title,
      }),
    );
    await reply({ customerID, threadID: created.id, text });
    return toThread(created);
  };

  const submitFeedback: Support["submitFeedback"] = async ({
    email,
    name,
    text,
    page,
    organizationKey,
  }) => {
    const address = email === "" ? ANONYMOUS.email : email;
    const { customer } = unwrap(
      await client.upsertCustomer({
        identifier: { emailAddress: address },
        onCreate: {
          fullName: name || (email === "" ? ANONYMOUS.name : email),
          email: { email: address, isVerified: false },
        },
        onUpdate: {},
      }),
    );
    const created = unwrap(
      await client.createThread({
        customerIdentifier: { customerId: customer.id },
        tenantIdentifier:
          organizationKey == null ? undefined : tenantOf(organizationKey),
        title: `Feedback on ${page}`,
        components: [{ componentText: { text } }],
      }),
    );
    return toThread(created);
  };

  return {
    ensureTenant,
    ensureMember,
    listThreads,
    retrieveThread,
    createThread,
    reply,
    submitFeedback,
  };
};
