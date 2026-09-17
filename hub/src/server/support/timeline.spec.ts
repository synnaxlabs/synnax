// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { type EntryInput, entryNodeZ, toMessages } from "@/server/support/timeline";

const AT = "2026-09-17T12:00:00.000Z";
const customer: EntryInput["actor"] = {
  __typename: "CustomerActor",
  customer: { fullName: "Gaal Dornik" },
};
const staff: EntryInput["actor"] = {
  __typename: "UserActor",
  user: { publicName: "Hari" },
};
const node = (
  entry: EntryInput["entry"],
  actor: EntryInput["actor"] = customer,
  id = "e1",
): EntryInput => ({
  id,
  timestamp: { iso8601: AT },
  actor,
  entry,
});
const messages = (nodes: EntryInput[]) => toMessages(entryNodeZ.array().parse(nodes));

describe("timeline.toMessages", () => {
  it("should map a customer chat", () => {
    expect(messages([node({ __typename: "ChatEntry", text: "hello" })])).toEqual([
      {
        id: "e1",
        at: new Date(AT),
        from: "customer",
        author: "Gaal Dornik",
        text: "hello",
      },
    ]);
  });

  it("should attribute a staff email to its author and prefer markdown", () => {
    const [m] = messages([
      node(
        { __typename: "EmailEntry", textContent: "plain", markdownContent: "**md**" },
        staff,
      ),
    ]);
    expect(m.from).toBe("staff");
    expect(m.author).toBe("Hari");
    expect(m.text).toBe("**md**");
  });

  it("should join the text components of a custom entry", () => {
    const [m] = messages([
      node({
        __typename: "CustomEntry",
        title: "Feedback",
        components: [
          { __typename: "ComponentText", text: "a" },
          { __typename: "ComponentSpacer" },
          { __typename: "ComponentPlainText", plainText: "b" },
        ],
      }),
    ]);
    expect(m.text).toBe("a\n\nb");
  });

  it("should fall back to the title of a custom entry without text", () => {
    const [m] = messages([
      node({ __typename: "CustomEntry", title: "Feedback", components: [] }),
    ]);
    expect(m.text).toBe("Feedback");
  });

  it("should drop entries that carry no customer-visible text", () => {
    expect(
      messages([
        node({ __typename: "NoteEntry" }, staff, "n"),
        node({ __typename: "ChatEntry", text: null }, customer, "c"),
        node(
          { __typename: "ThreadStatusTransitionedEntry" },
          { __typename: "SystemActor" },
        ),
      ]),
    ).toEqual([]);
  });

  it("should mark machine and system actors as system", () => {
    const [m] = messages([
      node(
        { __typename: "ChatEntry", text: "auto" },
        { __typename: "MachineUserActor" },
      ),
    ]);
    expect(m.from).toBe("system");
    expect(m.author).toBeNull();
  });
});
