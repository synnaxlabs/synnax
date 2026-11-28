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
  parseHexBytes,
} from "@/platform/icd/FormattedField";

const QUERY_ERROR = "Query must be pairs of hex digits, such as 01 03";

const RATE_INPUT_PROPS = { endContent: "Hz" } as const;

export interface PollFieldsProps {
  path: deep.Path<icd.ReadEntry>;
}

/**
 * The query that polls the device of an entry, and its rate. A query turns the poll on,
 * and clearing it turns the poll off, so the poll is never stored without a query.
 */
export const PollFields = ({ path }: PollFieldsProps): ReactElement => {
  const poll = Form.useFieldValue(path.poll, { optional: true });
  return (
    <Flex.Box x>
      <FormattedField<icd.Poll>
        path={path.poll}
        label="Query"
        placeholder="None"
        optional
        // The task cache holds config as the Core sent it, so the query can be base64.
        format={({ query }) => formatHexBytes(binary.bytesZ.parse(query))}
        parse={(text) => {
          const query = parseHexBytes(text);
          if (query == null) return { error: QUERY_ERROR };
          return { value: icd.pollZ.parse({ query, rate: poll?.rate }) };
        }}
      />
      {poll != null && (
        <Form.NumericField
          path={deep.present(path.poll).rate}
          label="Rate"
          inputProps={RATE_INPUT_PROPS}
        />
      )}
    </Flex.Box>
  );
};
