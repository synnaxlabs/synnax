// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { asc, desc, eq, inArray } from "drizzle-orm";

import { type Store } from "@/server/db/db";
import {
  event,
  license,
  type Message,
  message,
  type Organization,
  organization,
  type Sender,
  type Thread,
  thread,
  type ThreadKind,
} from "@/server/db/schema";
import { type Mailer } from "@/server/mail";
import { comment, description, replyMail } from "@/server/support/format";
import { type Status, type Tracker } from "@/server/support/tracker";

export interface OpenArgs {
  kind: ThreadKind;
  organization: Organization | null;
  title: string;
  text: string;
  author: string;
  contact: string;
  createdBy: string;
  page?: string;
  site: string;
  now: Date;
}

/**
 * open records a thread and its first message, files the Linear issue with the
 * organization's licenses in its body, and attaches a customer request from the
 * organization's Linear customer, created on first use.
 */
export const open = async (
  store: Store,
  tracker: Tracker,
  args: OpenArgs,
): Promise<Thread> => {
  const { kind, organization: org, title, text, author, contact, createdBy } = args;
  const licenses =
    org == null
      ? []
      : await store.query
          .select()
          .from(license)
          .where(eq(license.organization, org.key))
          .orderBy(desc(license.issuedAt));
  const customerID = org == null ? undefined : await customerOf(store, tracker, org);
  const issue = await tracker.createIssue({
    title,
    description: description({ ...args, organization: org, licenses }),
    customerID,
    need: text,
  });
  const [row] = await store.query
    .insert(thread)
    .values({
      kind,
      organization: org?.key,
      title,
      contact,
      createdBy,
      issueID: issue.id,
      issueIdentifier: issue.identifier,
      issueURL: issue.url,
      createdAt: args.now,
      updatedAt: args.now,
    })
    .returning();
  await store.query
    .insert(message)
    .values({ thread: row.key, sender: "customer", author, text, at: args.now });
  await store.query.insert(event).values({
    kind: kind === "feedback" ? "feedback" : "thread",
    actor: createdBy === "" ? contact : createdBy,
    organization: org?.key,
    detail: { thread: row.key, issue: issue.identifier },
  });
  return row;
};

const customerOf = async (
  store: Store,
  tracker: Tracker,
  org: Organization,
): Promise<string> => {
  if (org.linearCustomerID != null) return org.linearCustomerID;
  const linearCustomerID = await tracker.ensureCustomer(org);
  await store.query
    .update(organization)
    .set({ linearCustomerID })
    .where(eq(organization.key, org.key));
  org.linearCustomerID = linearCustomerID;
  return linearCustomerID;
};

export type Listed = Thread & { status: Status };

/** list returns an organization's threads, newest first, with their issue status. */
export const list = async (
  store: Store,
  tracker: Tracker,
  organizationKey: string,
): Promise<Listed[]> => {
  const rows = await store.query
    .select()
    .from(thread)
    .where(eq(thread.organization, organizationKey))
    .orderBy(desc(thread.updatedAt));
  const statuses = await tracker.statuses(rows.map((t) => t.issueID));
  return rows.map((t) => ({ ...t, status: statuses.get(t.issueID) ?? "open" }));
};

export interface Retrieved {
  thread: Thread;
  status: Status;
  messages: Message[];
}

export const retrieve = async (
  store: Store,
  tracker: Tracker,
  key: string,
): Promise<Retrieved | null> => {
  const [row] = await store.query.select().from(thread).where(eq(thread.key, key));
  if (row == null) return null;
  const [messages, statuses] = await Promise.all([
    store.query
      .select()
      .from(message)
      .where(eq(message.thread, key))
      .orderBy(asc(message.key)),
    tracker.statuses([row.issueID]),
  ]);
  return { thread: row, status: statuses.get(row.issueID) ?? "open", messages };
};

export interface ReplyArgs {
  thread: Thread;
  status: Status;
  sender: Sender;
  author: string;
  actor: string;
  text: string;
  site: string;
  now: Date;
}

/**
 * reply appends a message, mirrors it onto the issue, reopens a done issue when a
 * member writes, and mails the thread's contact when staff write.
 */
export const reply = async (
  store: Store,
  tracker: Tracker,
  mail: Mailer,
  { thread: t, status, sender, author, actor, text, site, now }: ReplyArgs,
): Promise<void> => {
  await tracker.comment(t.issueID, comment(sender, author, text));
  if (sender === "customer" && status === "done") await tracker.reopen(t.issueID);
  await store.query
    .insert(message)
    .values({ thread: t.key, sender, author, text, at: now });
  await store.query.update(thread).set({ updatedAt: now }).where(eq(thread.key, t.key));
  await store.query.insert(event).values({
    kind: "thread",
    actor,
    organization: t.organization,
    detail: { thread: t.key, reply: true },
  });
  if (sender === "staff" && t.contact !== "")
    await mail.send({
      to: [t.contact],
      ...replyMail({ title: t.title, author, text, url: `${site}/support/${t.key}` }),
    });
};

/** listForOrganizations returns threads across organizations, for staff. */
export const listForOrganizations = async (
  store: Store,
  tracker: Tracker,
  keys: string[],
): Promise<Listed[]> => {
  if (keys.length === 0) return [];
  const rows = await store.query
    .select()
    .from(thread)
    .where(inArray(thread.organization, keys))
    .orderBy(desc(thread.updatedAt));
  const statuses = await tracker.statuses(rows.map((t) => t.issueID));
  return rows.map((t) => ({ ...t, status: statuses.get(t.issueID) ?? "open" }));
};
