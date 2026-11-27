// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { icd } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { ICD } from "@/platform/icd";

interface Message {
  match?: icd.JSONMatch;
  pointers?: string[];
}

describe("route", () => {
  const present = (pointer: string): Message => ({
    match: { type: "present", pointer },
  });
  const kind = (value: string, ...pointers: string[]): Message => ({
    match: { type: "string", pointer: "/kind", value },
    pointers,
  });
  const fields = (...pointers: string[]) => pointers.map((pointer) => ({ pointer }));
  const route = (pointers: string[], messages: Message[]) =>
    ICD.route(fields(...pointers), messages, (m) => m.pointers ?? []);

  it("should give the message whose present rule is at a field", () => {
    const rpm = present("/rpm");
    expect(route(["/t", "/rpm"], [present("/fault"), rpm])).toEqual({ message: rpm });
  });

  it("should give the message whose present rule holds a field", () => {
    const status = present("/status");
    expect(route(["/status/code"], [status])).toEqual({ message: status });
    expect(route(["/statuses"], [status])).toEqual({});
  });

  it("should give the message with no rule when no present rule matches", () => {
    const rest = {};
    expect(route(["/t"], [present("/rpm"), rest])).toEqual({ message: rest });
  });

  it("should give no message when none matches and each has a rule", () => {
    expect(route(["/t"], [present("/rpm")])).toEqual({});
  });

  it("should give the message of a value rule that reads the most of the fields", () => {
    const status = kind("status", "/kind", "/code");
    const reading = kind("reading", "/kind", "/temp", "/rpm");
    expect(route(["/kind", "/temp", "/rpm"], [status, reading])).toEqual({
      message: reading,
    });
  });

  it("should give the message with no rule when it reads the most of the fields", () => {
    const rest = { pointers: ["/kind", "/temp"] };
    expect(route(["/kind", "/temp"], [kind("status", "/kind"), rest])).toEqual({
      message: rest,
    });
  });

  it("should split when two messages read the most of the fields", () => {
    const messages = [kind("status", "/kind", "/code"), kind("fault", "/kind", "/rpm")];
    expect(route(["/kind", "/temp"], messages)).toEqual({ split: "/kind" });
  });

  it("should leave a value rule out when no field is at its pointer", () => {
    const rest = {};
    expect(route(["/temp"], [kind("status", "/temp"), rest])).toEqual({
      message: rest,
    });
  });
});
