// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { icd } from "@synnaxlabs/client";
import type { payload } from "@synnaxlabs/x";

/** @returns an empty payload of the format, with no fields. */
export const createPayload = (format: payload.Format): icd.Payload => {
  switch (format) {
    case "binary":
      return { format, fields: [] };
    case "text":
      return { format, delimiter: ",", prefix: "", fields: [] };
    case "json":
      return { format, fields: [] };
  }
};

/** The kind that a task tree shows for a message of each format. */
export const FORMAT_KINDS: Record<payload.Format, string> = {
  json: "JSON",
  text: "TEXT",
  binary: "BINARY",
};
