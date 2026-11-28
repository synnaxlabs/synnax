// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd } from "@synnaxlabs/client";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Access, ICD as PICD } from "@synnaxlabs/pluto";
import { unique } from "@synnaxlabs/x";
import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  type ReactNode,
  useContext,
  useEffect,
  useState,
} from "react";

import { fieldNames } from "@/platform/icd/fieldNames";
import { fieldsPath, jsonFields, type Messages } from "@/platform/icd/read";
import { Task } from "@/platform/task";

/** The form that edits an ICD. */
export type FormatForm = ReturnType<typeof PICD.useForm>["form"];

/** The open forms of the ICDs that a read task reads, by ICD key. */
export type Formats = ReadonlyMap<icd.Key, FormatForm>;

const Context = createContext<Formats>(new Map());
Context.displayName = "ICD.FormatsContext";

/** @returns the forms that an enclosing WithFormats opens. */
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

/** @returns what read takes from formats, current with each edit of a form. */
const useCurrent = <T,>(formats: Formats, read: (formats: Formats) => T): T => {
  const [value, setValue] = useState(() => read(formats));
  useEffect(() => {
    const update = () => setValue(read(formats));
    update();
    const destructors = [...formats.values()].map((form) => form.bind(update));
    return () => destructors.forEach((destroy) => destroy());
  }, [formats, read]);
  return value;
};

/** @returns the messages of each open ICD, by ICD key, current with each edit. */
export const useMessages = (): Messages => useCurrent(useFormats(), messagesOf);

const valuesOf = (formats: Formats): Pick<icd.ICD, "key" | "name" | "messages">[] =>
  [...formats].map(([key, form]) => ({ key, ...form.value() }));

/** @returns each ICD that formats open, current with each edit. */
export const useValues = (
  formats: Formats,
): Pick<icd.ICD, "key" | "name" | "messages">[] => useCurrent(formats, valuesOf);

/**
 * @returns a message of an ICD that an enclosing WithFormats opens, current with each
 * edit, or null when the ICD has no message with the key.
 */
export const useMessage = (
  icdKey: icd.Key,
  message: icd.MessageKey,
): icd.Message | null =>
  PForm.useFieldValue(PICD.formPath.messages[message], {
    optional: true,
    ctx: formatOf(useFormats(), icdKey),
  });

/** @returns a message of an open ICD, or undefined when the ICD has none. */
export const heldMessage = (
  formats: Formats,
  icdKey: icd.Key,
  message: icd.MessageKey,
): icd.Message | undefined =>
  formats
    .get(icdKey)
    ?.value()
    .messages.find(({ key }) => key === message);

/** Inserts entries into the lists of one ICD. */
export type Inserter = Pick<FormatForm, "insert">;

/** @returns an inserter that adds each insert to actions, for a dispatch to an ICD. */
export const actionInserter = (actions: icd.Action[]): Inserter => ({
  insert: (path, value, after) =>
    actions.push(icd.insert({ path: String(path).split("."), value, after })),
});

/**
 * Appends each field whose pointer the message lacks.
 * @returns the fields it appends.
 */
export const appendFields = (
  to: Inserter,
  message: icd.Message,
  fields: icd.JSONField[],
): icd.JSONField[] => {
  const held = jsonFields(message);
  const pointers = new Set(held.map(({ pointer }) => pointer));
  const added = fields.filter(({ pointer }) => !pointers.has(pointer));
  const names = fieldNames(
    added.map(({ pointer }) => pointer),
    held.map(({ name }) => name),
  );
  const appended = added.map((field, i) => ({ ...field, name: names[i] }));
  let after = held.at(-1)?.key;
  for (const field of appended) {
    to.insert(fieldsPath(message.key), field, after);
    after = field.key;
  }
  return appended;
};

interface WithFormatProps extends PropsWithChildren {
  icdKey: icd.Key;
}

const WithFormat = ({ icdKey, children }: WithFormatProps): ReactElement => {
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

export interface WithFormatsProps extends PropsWithChildren {
  /** The ICDs to open. A key may repeat. */
  icdKeys: icd.Key[];
}

/**
 * Opens the form of each ICD in icdKeys, for useFormats in children. Each edit updates
 * the ICD at once, and deploying the task flushes pending edits first.
 */
export const WithFormats = ({ icdKeys, children }: WithFormatsProps): ReactElement => (
  <>
    {unique.unique(icdKeys).reduceRight<ReactNode>(
      (child, key) => (
        <WithFormat key={key} icdKey={key}>
          {child}
        </WithFormat>
      ),
      children,
    )}
  </>
);
