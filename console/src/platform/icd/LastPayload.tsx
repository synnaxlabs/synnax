// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/icd/LastPayload.css";

import type { icd, Synnax } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { useDebouncedCallback } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import type { Flux } from "@synnaxlabs/pluto";
import { type payload, TimeSpan, type TimeStamp } from "@synnaxlabs/x";
import { Fragment, type ReactElement, useEffect, useState } from "react";

import { CSS } from "@/platform/css";
import { useListen } from "@/platform/icd/picker/useListen";

/** The last message of a source: when it arrived, its format, and its payload. */
export interface Sample {
  time: TimeStamp;
  format: payload.Format;
  /** The start of the payload: text when format is json or text, else hex digits. */
  payload: string;
}

const BYTES_PER_ROW = 8;
// The Core runs one scan command at a time, so a listen per keystroke would queue.
const SOURCE_DEBOUNCE = TimeSpan.milliseconds(500);
const BYTE_LIMIT = 256;
const PART_LIMIT = 64;
// The characters that a text value loses from each end, as the Core trims it.
const TRIMMED = /^[ \t\r]+|[ \t\r]+$/g;

const trim = (text: string): string => text.replace(TRIMMED, "");

/**
 * @returns the tag that each part shows: the longest tag that starts it, on the first
 * part that the tag claims. A tagged field reads only that part.
 */
const tagsOf = (parts: string[], tags: string[]): (string | undefined)[] => {
  const claimed = new Set<string>();
  return parts.map((part) => {
    const text = trim(part);
    const tag = tags
      .filter((t) => text.startsWith(t))
      .reduce<string | undefined>(
        (a, b) => (a == null || b.length > a.length ? b : a),
        undefined,
      );
    if (tag == null || claimed.has(tag)) return undefined;
    claimed.add(tag);
    return tag;
  });
};

const Note = ({ children }: { children: string }): ReactElement => (
  <Text.Text level="small" color={9}>
    {children}
  </Text.Text>
);

interface PartsProps {
  line: string;
  payload: icd.TextPayload;
  /** The key of the field whose part to mark. */
  field?: icd.FieldKey;
}

const Parts = ({ line, payload, field }: PartsProps) => {
  const parts = line.split(payload.delimiter);
  const tags = tagsOf(
    parts,
    payload.fields.flatMap((f) => (f.encoding === "tagged" ? [f.tag] : [])),
  );
  const marked = payload.fields.find(({ key }) => key === field);
  let position = -1;
  let value = "";
  if (marked?.encoding === "delimited" && marked.position < parts.length) {
    position = marked.position;
    value = trim(parts[position]);
  } else if (marked?.encoding === "tagged") {
    position = tags.indexOf(marked.tag);
    if (position >= 0) value = trim(trim(parts[position]).slice(marked.tag.length));
  }
  return (
    <>
      <Flex.Box x wrap className={CSS.BE("icd-last-payload", "parts")}>
        {parts.slice(0, PART_LIMIT).map((part, i) => (
          <Flex.Box
            y
            gap="tiny"
            key={i}
            className={CSS.cls(
              CSS.BE("icd-last-payload", "part"),
              i === position && CSS.M("marked"),
            )}
          >
            <Text.Text level="small" variant="code">
              {part}
            </Text.Text>
            <Text.Text level="small" color={8}>
              {tags[i] == null ? i : `${i} · ${tags[i]}`}
            </Text.Text>
          </Flex.Box>
        ))}
      </Flex.Box>
      {parts.length > PART_LIMIT && (
        <Note>{`Shows the first ${PART_LIMIT} parts.`}</Note>
      )}
      {value !== "" && <Note>{`Last value ${value}`}</Note>}
    </>
  );
};

const Bytes = ({ hex }: { hex: string }) => {
  const bytes = hex.toUpperCase().match(/../g) ?? [];
  const shown = bytes.slice(0, BYTE_LIMIT);
  const rows = Array.from({ length: Math.ceil(shown.length / BYTES_PER_ROW) }, (_, r) =>
    shown.slice(r * BYTES_PER_ROW, (r + 1) * BYTES_PER_ROW),
  );
  return (
    <>
      <div className={CSS.BE("icd-last-payload", "bytes")}>
        {rows.map((row, r) => (
          <Fragment key={r}>
            <Text.Text level="small" color={8}>
              {r * BYTES_PER_ROW}
            </Text.Text>
            {row.map((byte, i) => (
              <Text.Text level="small" variant="code" key={i}>
                {byte}
              </Text.Text>
            ))}
          </Fragment>
        ))}
      </div>
      {bytes.length > BYTE_LIMIT && (
        <Note>{`Shows the first ${BYTE_LIMIT} bytes.`}</Note>
      )}
    </>
  );
};

interface PayloadProps extends Pick<PartsProps, "field"> {
  payload: icd.TextPayload | icd.BinaryPayload;
  sample: Sample;
}

const Payload = ({ payload, sample, field }: PayloadProps): ReactElement => {
  if (sample.payload === "") return <Note>The last payload is empty.</Note>;
  if (sample.format === "binary") return <Bytes hex={sample.payload} />;
  if (payload.format === "text")
    return <Parts line={sample.payload} payload={payload} field={field} />;
  return (
    <Text.Text
      level="small"
      variant="code"
      className={CSS.BE("icd-last-payload", "text")}
    >
      {sample.payload}
    </Text.Text>
  );
};

export interface UseLastPayloadArgs {
  /** What the payload belongs to, such as a task entry. A new key listens at once. */
  key: string;
  /**
   * Names the source, such as a topic, in a failure. A new source listens once it stops
   * changing. An empty source listens to nothing.
   */
  source: string;
  /**
   * Listens to the source.
   * @returns its last message, or null when it finds none.
   */
  sample: (client: Synnax) => Promise<Sample | null>;
}

export interface LastPayloadState {
  /** The latest listen. One in flight keeps the answer of the one before. */
  result: Flux.Result<{ sample: Sample | null }>;
  /** True while the source changes, before it listens. */
  settling: boolean;
  /** Listens again. */
  listen: () => void;
}

/**
 * Listens for the last payload of a source on mount, on a new key, once the source
 * stops changing, and on listen. Call it above the panes that show the source and its
 * fields, so that they share one listen.
 */
export const useLastPayload = ({
  key,
  source,
  sample,
}: UseLastPayloadArgs): LastPayloadState => {
  const [settled, setSettled] = useState({ key, source });
  if (settled.key !== key) setSettled({ key, source });
  const settle = useDebouncedCallback(
    (next: string) => setSettled({ key, source: next }),
    SOURCE_DEBOUNCE,
    [key],
  );
  useEffect(() => {
    settle(source);
    return () => settle.cancel();
  }, [source, settle]);
  const { result, listen } = useListen<{ sample: Sample | null }>(
    async (client) => ({
      sample: settled.source === "" ? null : await sample(client),
    }),
    { message: "Failed to read the last payload", name: settled.source },
    [settled.key, settled.source],
  );
  return {
    result,
    settling: settled.key !== key || settled.source !== source,
    listen,
  };
};

export interface LastPayloadProps extends Pick<PartsProps, "field"> {
  /** The listen that useLastPayload returns. */
  state: LastPayloadState;
  /** The payload of the message that the source sends. */
  payload: icd.TextPayload | icd.BinaryPayload;
  /** Says what a listen does while it runs, such as "Listening for 3 seconds". */
  pendingMessage: string;
  /** Says that a listen found no message. */
  emptyMessage: string;
}

/**
 * The last payload of a source, laid out as its message reads it: the parts of a text
 * line with their positions, or the bytes of a binary payload with their offsets. A
 * text field marks the part that it reads and shows its value.
 */
export const LastPayload = ({
  state: { result, settling, listen },
  payload,
  field,
  pendingMessage,
  emptyMessage,
}: LastPayloadProps): ReactElement => {
  let body: ReactElement;
  if (result.variant === "error") body = <Status.Summary status={result.status} />;
  else if (result.variant === "loading" || result.data == null || settling)
    body = <Note>{pendingMessage}</Note>;
  else if (result.data.sample == null) body = <Note>{emptyMessage}</Note>;
  else body = <Payload payload={payload} sample={result.data.sample} field={field} />;
  return (
    <PForm.Section
      title="Last payload"
      actions={
        <Button.Button
          variant="text"
          size="small"
          onClick={listen}
          disabled={result.variant === "loading"}
        >
          <Icon.Refresh />
          Listen again
        </Button.Button>
      }
    >
      {body}
    </PForm.Section>
  );
};
