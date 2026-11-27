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
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import type { payload, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { Fragment, type ReactElement } from "react";

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
const BYTE_LIMIT = 256;
const PART_LIMIT = 64;

const Note = ({ children }: { children: string }): ReactElement => (
  <Text.Text level="small" color={9}>
    {children}
  </Text.Text>
);

const Parts = ({ line, delimiter }: { line: string; delimiter: string }) => {
  const parts = line.split(delimiter);
  return (
    <>
      <Flex.Box x wrap className={CSS.BE("icd-last-payload", "parts")}>
        {parts.slice(0, PART_LIMIT).map((part, position) => (
          <Flex.Box y gap="tiny" key={position}>
            <Text.Text level="small" variant="code">
              {part}
            </Text.Text>
            <Text.Text level="small" color={8}>
              {position}
            </Text.Text>
          </Flex.Box>
        ))}
      </Flex.Box>
      {parts.length > PART_LIMIT && (
        <Note>{`Shows the first ${PART_LIMIT} parts.`}</Note>
      )}
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

interface PayloadProps {
  payload: icd.TextPayload | icd.BinaryPayload;
  sample: Sample;
}

const Payload = ({ payload, sample }: PayloadProps): ReactElement => {
  if (sample.format === "binary") return <Bytes hex={sample.payload} />;
  if (payload.format === "text")
    return <Parts line={sample.payload} delimiter={payload.delimiter} />;
  return (
    <Text.Text level="small" variant="code">
      {sample.payload}
    </Text.Text>
  );
};

export interface LastPayloadProps {
  /** The payload of the message that the source sends. */
  payload: icd.TextPayload | icd.BinaryPayload;
  /** Names the source, such as a topic, in the text for a source that sends nothing. */
  source: string;
  /** How long a listen lasts. */
  duration: TimeSpan;
  /**
   * Listens to the source.
   * @returns its last message, or null when it sends none while it listens.
   */
  sample: (client: Synnax) => Promise<Sample | null>;
}

/**
 * The last payload of a source, laid out as its message reads it: the parts of a text
 * line with their positions, or the bytes of a binary payload with their offsets.
 * Listens on mount, when the source changes, and on Listen again.
 */
export const LastPayload = ({
  payload,
  source,
  duration,
  sample,
}: LastPayloadProps): ReactElement => {
  const { result, listen } = useListen<{ sample: Sample | null }>(
    async (client) => ({ sample: await sample(client) }),
    { message: "Failed to read the last payload", name: source },
    [source],
  );
  let body: ReactElement;
  if (result.variant === "error") body = <Status.Summary status={result.status} />;
  else if (result.data == null)
    body = <Note>{`Listening for ${duration.seconds} seconds`}</Note>;
  else if (result.data.sample == null)
    body = <Note>{`No message on ${source} in ${duration.seconds} seconds.`}</Note>;
  else body = <Payload payload={payload} sample={result.data.sample} />;
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
