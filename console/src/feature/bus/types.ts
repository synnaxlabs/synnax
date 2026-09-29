// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type bus, type library } from "@synnaxlabs/client";
import { type z } from "zod";

/** A message entry of a task's config: the fields a read or write task binds. */
export type Message = bus.ReadMessage | bus.WriteMessage;

/** Whether an integration can carry a library message. */
export interface Accepts {
  (message: library.MessageEntry): boolean;
}

/** Accepts the messages a byte stream can carry: no identifier, or a field or token. */
export const acceptsStream: Accepts = ({ identifier }) =>
  identifier == null || identifier.type === "field" || identifier.type === "token";

/** @returns the message entries of a library, keyed by entry key. */
export const messagesOf = (
  lib: library.Library | undefined,
): Map<library.EntryKey, library.MessageEntry> => {
  const out = new Map<library.EntryKey, library.MessageEntry>();
  lib?.entries.forEach((e) => {
    if (e.kind === "message") out.set(e.key, e);
  });
  return out;
};

type Issues = z.core.ParsePayload["issues"];

const push = (issues: Issues, input: unknown, path: PropertyKey[], message: string) =>
  issues.push({ code: "custom", message, path, input });

const validateDevice = (issues: Issues, input: unknown, device: string) => {
  if (device === "") push(issues, input, ["device"], "Device is required");
};

const validateMessages = (issues: Issues, input: unknown, messages: Message[]) => {
  if (messages.every((m) => m.disabled))
    push(issues, input, ["messages"], "Add at least one enabled message");
};

/** Rejects a read config the Driver cannot run. */
export const validateRead = ({
  value,
  issues,
}: z.core.ParsePayload<bus.ReadConfig>) => {
  validateDevice(issues, value, value.device);
  validateMessages(issues, value, value.messages);
  value.messages.forEach((m, i) => {
    if (!m.disabled && m.fields.length === 0)
      push(issues, value, ["messages", i, "fields"], "Select at least one field");
  });
};

/** Rejects a write config the Driver cannot run. */
export const validateWrite = ({
  value,
  issues,
}: z.core.ParsePayload<bus.WriteConfig>) => {
  validateDevice(issues, value, value.device);
  validateMessages(issues, value, value.messages);
};

/** Rejects framing the Driver cannot build. */
export const validateFraming = ({
  value: { framing },
  issues,
}: z.core.ParsePayload<{ framing: bus.Framing }>) => {
  const path = (field: string) => ["framing", field];
  switch (framing.type) {
    case "delimiter":
      if (framing.delimiter === "")
        push(issues, framing, path("delimiter"), "Delimiter is required");
      break;
    case "fixed":
      if (framing.length === 0)
        push(issues, framing, path("length"), "Length must be at least 1");
      break;
    case "sync":
      if (!/^([0-9a-fA-F]{2})+$/.test(framing.sync))
        push(issues, framing, path("sync"), "Sync must be hex bytes, such as AA55");
      break;
    default:
  }
};

/** Rejects a poll rate or timeout the Driver cannot run. */
export const validatePoll = ({
  value: { rate, timeout },
  issues,
}: z.core.ParsePayload<bus.PollConfig>) => {
  if (rate <= 0) push(issues, rate, ["rate"], "Rate must be greater than 0");
  if (timeout.valueOf() <= 0n)
    push(issues, timeout, ["timeout"], "Timeout must be greater than 0");
};
