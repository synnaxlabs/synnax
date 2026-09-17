// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

export const SENDERS = ["customer", "staff", "system"] as const;
export type Sender = (typeof SENDERS)[number];

/** Message is one customer-visible entry of a thread, oldest first. */
export interface Message {
  id: string;
  at: Date;
  from: Sender;
  /** author is the display name Plain attributes the entry to, when it has one. */
  author: string | null;
  text: string;
}

const dateTimeZ = z.object({ iso8601: z.string() });

const actorZ = z.discriminatedUnion("__typename", [
  z.object({
    __typename: z.literal("UserActor"),
    user: z.object({ publicName: z.string() }),
  }),
  z.object({
    __typename: z.literal("CustomerActor"),
    customer: z.object({ fullName: z.string() }),
  }),
  z.object({ __typename: z.literal("DeletedCustomerActor") }),
  z.object({ __typename: z.literal("MachineUserActor") }),
  z.object({ __typename: z.literal("SystemActor") }),
]);

/** OTHER stands in for any entry or component kind the portal does not render. */
const OTHER = { __typename: "Other" as const };
const otherZ = z.object({ __typename: z.string() }).transform(() => OTHER);

const componentZ = z.union([
  z.object({ __typename: z.literal("ComponentText"), text: z.string() }),
  z.object({ __typename: z.literal("ComponentPlainText"), plainText: z.string() }),
  otherZ,
]);

const entryZ = z.union([
  z.object({ __typename: z.literal("ChatEntry"), text: z.string().nullable() }),
  z.object({
    __typename: z.literal("EmailEntry"),
    textContent: z.string().nullable(),
    markdownContent: z.string().nullable(),
  }),
  z.object({
    __typename: z.literal("CustomEntry"),
    title: z.string(),
    components: z.array(componentZ),
  }),
  otherZ,
]);

export const entryNodeZ = z.object({
  id: z.string(),
  timestamp: dateTimeZ,
  actor: actorZ,
  entry: entryZ,
});
export type EntryNode = z.output<typeof entryNodeZ>;
export type EntryInput = z.input<typeof entryNodeZ>;

const text = (entry: EntryNode["entry"]): string | null => {
  switch (entry.__typename) {
    case "ChatEntry":
      return entry.text;
    case "EmailEntry":
      return entry.markdownContent ?? entry.textContent;
    case "CustomEntry": {
      const parts = entry.components.flatMap((c) => {
        if (c.__typename === "ComponentText") return [c.text];
        if (c.__typename === "ComponentPlainText") return [c.plainText];
        return [];
      });
      return parts.length === 0 ? entry.title : parts.join("\n\n");
    }
    default:
      return null;
  }
};

const sender = (actor: EntryNode["actor"]): Sender => {
  switch (actor.__typename) {
    case "CustomerActor":
    case "DeletedCustomerActor":
      return "customer";
    case "UserActor":
      return "staff";
    default:
      return "system";
  }
};

const author = (actor: EntryNode["actor"]): string | null => {
  switch (actor.__typename) {
    case "CustomerActor":
      return actor.customer.fullName;
    case "UserActor":
      return actor.user.publicName;
    default:
      return null;
  }
};

/**
 * toMessages keeps the entries a customer may read: chats, emails, and custom
 * entries. Internal notes and status changes carry no text and are dropped.
 */
export const toMessages = (nodes: EntryNode[]): Message[] =>
  nodes.flatMap((node) => {
    const body = text(node.entry);
    if (body == null || body === "") return [];
    return [
      {
        id: node.id,
        at: new Date(node.timestamp.iso8601),
        from: sender(node.actor),
        author: author(node.actor),
        text: body,
      },
    ];
  });
