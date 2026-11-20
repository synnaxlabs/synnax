// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd } from "@synnaxlabs/client";
import { Access, ICD as PICD } from "@synnaxlabs/pluto";
import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  useContext,
  useEffect,
  useState,
} from "react";

import { jsonFields, type Messages } from "@/platform/icd/read";
import { Task } from "@/platform/task";

/** The form that edits an ICD. */
export type FormatForm = ReturnType<typeof PICD.useForm>["form"];

/** The open forms of the ICDs that a read task reads, by ICD key. */
export type Formats = ReadonlyMap<icd.Key, FormatForm>;

const Context = createContext<Formats>(new Map());
Context.displayName = "ICD.FormatsContext";

/** @returns the forms that each enclosing WithFormat opens. */
export const useFormats = (): Formats => useContext(Context);

/** @returns the open form of an ICD that the task reads. */
export const formatOf = (formats: Formats, key: icd.Key): FormatForm => {
  const format = formats.get(key);
  if (format == null) throw new Error(`The form of ICD ${key} is not open`);
  return format;
};

/** @returns the messages of each open ICD, by ICD key. */
export const messagesOf = (formats: Formats): Messages =>
  new Map([...formats].map(([key, form]) => [key, form.value().messages]));

/** @returns the messages of each open ICD, by ICD key, current with each edit. */
export const useMessages = (): Messages => {
  const formats = useFormats();
  const [messages, setMessages] = useState(() => messagesOf(formats));
  useEffect(() => {
    const update = () => setMessages(messagesOf(formats));
    update();
    const destructors = [...formats.values()].map((form) => form.bind(update));
    return () => destructors.forEach((destroy) => destroy());
  }, [formats]);
  return messages;
};

/** @returns the fields of a message of an ICD. */
export const fieldsOf = (
  formats: Formats,
  icdKey: icd.Key,
  message: icd.MessageKey,
): icd.JSONField[] =>
  jsonFields(
    formats
      .get(icdKey)
      ?.value()
      .messages.find(({ key }) => key === message),
  );

export interface WithFormatProps extends PropsWithChildren {
  icdKey: icd.Key;
}

/**
 * Opens the form of an ICD that a read task reads, for useFormats in children. Each
 * edit updates the ICD at once, and deploying the task flushes pending edits first.
 */
export const WithFormat = ({ icdKey, children }: WithFormatProps): ReactElement => {
  const isPreview = Task.useIsPreview();
  const canEdit = Access.useUpdateGranted(icd.resourceID(icdKey));
  const { form, flush } = PICD.useForm({
    query: { key: icdKey },
    mode: isPreview || !canEdit ? "preview" : "normal",
  });
  PICD.useCheck({ key: icdKey, form });
  Task.useFlushFirst(flush);
  const formats = new Map(useContext(Context)).set(icdKey, form);
  return <Context value={formats}>{children}</Context>;
};
