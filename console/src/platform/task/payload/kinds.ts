// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CrudeDataType, type payload } from "@synnaxlabs/x";

import { kindOf } from "@/platform/task/payload/kind";

/**
 * Returns the kinds a channel field can write a sample of the data type as. The first
 * is the data type's own kind.
 */
export const channelKinds = (dataType: CrudeDataType): payload.Kind[] => {
  const kind = kindOf(dataType);
  switch (kind) {
    case "boolean":
      return ["boolean", "unsigned", "string"];
    case "float":
    case "integer":
    case "unsigned":
      return [kind, "string", "boolean"];
    default:
      return [kind];
  }
};

const KIND_NAMES: Record<payload.Kind, string> = {
  float: "Number",
  integer: "Number",
  unsigned: "Number",
  string: "String",
  boolean: "Boolean",
  timestamp: "Timestamp",
  json: "JSON",
  uuid: "UUID",
  bytes: "Base64",
};

/** Returns the name of the kind in a select. */
export const kindName = (kind: payload.Kind): string => KIND_NAMES[kind];
