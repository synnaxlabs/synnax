// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export type NodeIdType = "Numeric" | "String" | "GUID" | "ByteString";

export interface NumericNodeID {
  namespaceIndex: number;
  identifierType: "Numeric";
  identifier: number;
}

export interface GeneralNodeId {
  namespaceIndex: number;
  identifierType: "String" | "GUID" | "ByteString";
  identifier: string;
}

export type NodeId = NumericNodeID | GeneralNodeId;

const NODE_ID_REGEX = /NS=(\d+);(I|S|G|B)=(.+)/;

export const parseNodeID = (nodeIdStr: string): NodeId | null => {
  const match = nodeIdStr.match(NODE_ID_REGEX);
  if (match === null) return null;

  const namespaceIndex = parseInt(match[1]);
  const typeCode = match[2];
  const identifier: string = match[3];

  let identifierType: NodeIdType;

  switch (typeCode) {
    case "I":
      identifierType = "Numeric";
      return { namespaceIndex, identifierType, identifier: parseInt(identifier) };
    case "S":
      identifierType = "String";
      break;
    case "G":
      identifierType = "GUID";
      break;
    case "B":
      identifierType = "ByteString";
      break;
    default:
      return null;
  }

  return { namespaceIndex, identifierType, identifier };
};

export const nodeIDToString = ({
  namespaceIndex,
  identifierType,
  identifier,
}: NodeId): string => {
  const prefix = `NS=${namespaceIndex};`;
  switch (identifierType) {
    case "Numeric":
      return `${prefix}I=${identifier}`;
    case "String":
    case "GUID":
    case "ByteString":
      return `${prefix}${identifierType.charAt(0)}=${identifier}`;
  }
};
