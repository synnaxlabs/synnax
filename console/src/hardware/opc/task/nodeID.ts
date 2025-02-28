// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export interface NodeId {
  namespaceIndex: number;
  identifierType: NodeIdType;
  identifier: string | number; // Strings for String, GUID, and ByteString types, number for Numeric
}

export const parseNodeId = (nodeIdStr: string): NodeId | null => {
  const regex = /NS=(\d+);(I|S|G|B)=(.+)/;
  const match = nodeIdStr.match(regex);

  if (match === null) return null;

  const namespaceIndex = parseInt(match[1]);
  const typeCode = match[2];
  const identifier = match[3];

  let identifierType: NodeIdType;

  switch (typeCode) {
    case "I":
      identifierType = "Numeric";
      return {
        namespaceIndex,
        identifierType,
        identifier: parseInt(identifier),
      };
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

export const nodeIdToString = (nodeId: NodeId): string => {
  const prefix = `NS=${nodeId.namespaceIndex};`;
  switch (nodeId.identifierType) {
    case "Numeric":
      return `${prefix}I=${nodeId.identifier}`;
    case "String":
    case "GUID":
    case "ByteString":
      return `${prefix}${nodeId.identifierType.charAt(0)}=${nodeId.identifier}`;
  }
};

type NodeIdType = "Numeric" | "String" | "GUID" | "ByteString";
