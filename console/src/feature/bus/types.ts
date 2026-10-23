// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type bus, type icd } from "@synnaxlabs/client";
import { type z } from "zod";

/** A message entry of a task's config: the fields a read or write task binds. */
export type Message = bus.ReadMessage | bus.WriteMessage;

/** Whether an integration can carry an ICD message. */
export interface Accepts {
  (message: icd.MessageEntry): boolean;
}

/**
 * Accepts the messages a byte stream can carry: text, or binary with no identifier or a
 * field identifier.
 */
export const acceptsStream: Accepts = ({ payload }) =>
  payload.format === "text" ||
  payload.identifier == null ||
  payload.identifier.type === "field";

/** @returns the message entries of an ICD, keyed by entry key. */
export const messagesOf = (
  ic: icd.ICD | undefined,
): Map<icd.EntryKey, icd.MessageEntry> => {
  const out = new Map<icd.EntryKey, icd.MessageEntry>();
  ic?.entries.forEach((e) => {
    if (e.kind === "message") out.set(e.key, e);
  });
  return out;
};

/**
 * @returns the field of a message entry with the key.
 * @throws {Error} if the message has no such field.
 */
export const fieldOf = (
  entry: icd.MessageEntry,
  key: icd.FieldKey,
): icd.BaseField => {
  const field = entry.payload.fields.find((f) => f.key === key);
  if (field == null) throw new Error(`Field ${key} is not in message ${entry.name}`);
  return field;
};

type Issues = z.core.ParsePayload["issues"];

const push = (issues: Issues, input: unknown, path: PropertyKey[], message: string) =>
  issues.push({ code: "custom", message, path, input });

const validateDevice = (issues: Issues, input: unknown, device: string) => {
  if (device === "") push(issues, input, ["device"], "Device is required");
};

const validateICD = (issues: Issues, input: unknown, key?: icd.Key) => {
  if (key == null) push(issues, input, ["icd"], "Select an ICD");
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
  validateICD(issues, value, value.icd);
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
  validateICD(issues, value, value.icd);
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
