// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { binary, deep } from "@synnaxlabs/x";
import type { ReactElement } from "react";

import {
  formatHexBytes,
  FormattedField,
  orError,
  type Parsed,
  parseHexBytes,
} from "@/platform/icd/FormattedField";

const QUERY_ERROR = "Query must be pairs of hex digits, such as 01 03";

const RATE_INPUT_PROPS = { endContent: "Hz" } as const;

export interface PollFieldsProps {
  path: deep.Path<icd.ReadEntry>;
}

/** @returns the query bytes of the text, or why it holds none. */
const parseQuery = (text: string): Parsed<Uint8Array> =>
  orError(parseHexBytes(text), QUERY_ERROR);

// The task cache holds config as the Core sent it, so the query can be base64.
const formatQuery = (query: Uint8Array): string =>
  formatHexBytes(binary.bytesZ.parse(query));

/**
 * The query that polls the device of an entry, and its rate. A query starts a poll at
 * the default rate, and clearing the query stops the poll.
 */
export const PollFields = ({ path }: PollFieldsProps): ReactElement => {
  const { set } = Form.useContext();
  const poll = Form.useFieldValue(path.poll, { optional: true });
  const present = deep.present(path.poll);
  return (
    <Flex.Box x>
      {poll == null ? (
        <FormattedField<icd.Poll>
          path={path.poll}
          label="Query"
          placeholder="None"
          optional
          format={({ query }) => formatQuery(query)}
          parse={(text) => {
            const parsed = parseQuery(text);
            if ("error" in parsed) return parsed;
            return { value: icd.pollZ.parse({ query: parsed.value }) };
          }}
        />
      ) : (
        <>
          <FormattedField<Uint8Array>
            path={present.query}
            label="Query"
            placeholder="None"
            optional
            onClear={() => set(path.poll, undefined)}
            format={formatQuery}
            parse={parseQuery}
          />
          <Form.NumericField
            path={present.rate}
            label="Rate"
            inputProps={RATE_INPUT_PROPS}
          />
        </>
      )}
    </Flex.Box>
  );
};
