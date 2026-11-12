// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import KINDS from "../../../../../arc/go/analyzer/codes/codes.gen.json";

interface Kind {
  severity: string;
  message: string;
}

/** The props that a `Problem.Item` takes from an Arc diagnostic kind. */
export interface Props {
  code: string;
  severity: "error" | "warning";
  message?: string | string[];
}

const fill = (code: string, format: string, values: string[]): string => {
  let used = 0;
  const message = format.replace(/%[a-z]/g, (verb) => {
    if (used === values.length) throw new Error(`${code} has a verb with no value`);
    const value = values[used++];
    return verb === "%q" ? `"${value}"` : value;
  });
  if (used !== values.length) throw new Error(`${code} has a value with no verb`);
  return message;
};

/**
 * Gives a `Problem.Item` the code, severity, and message of an Arc diagnostic kind.
 * @param code - The code of the kind, for example `ARC3001`.
 * @param values - For each message to show, the text that stands for each verb of the
 * message format, in order. With no list, the format has no verbs and shows as it is.
 * @returns The props. They have no message when the kind takes its message from an
 * error.
 * @throws If no kind has the code, or the values do not match the verbs.
 */
export const diagnostic = (code: string, ...values: string[][]): Props => {
  const kind: Kind | undefined = (KINDS as Record<string, Kind>)[code];
  if (kind == null) throw new Error(`no Arc diagnostic kind has the code ${code}`);
  const props: Props = { code, severity: kind.severity as Props["severity"] };
  if (kind.message === "") return props;
  const messages = (values.length === 0 ? [[]] : values).map((v) =>
    fill(code, kind.message, v),
  );
  return { ...props, message: messages.length === 1 ? messages[0] : messages };
};
