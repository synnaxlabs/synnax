// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { nodeIDToString, parseNodeID } from "@/hardware/opc/task/nodeID";

describe("parseNodeID", () => {
  it("should parse a numeric node ID", () => {
    const nodeId = parseNodeID("NS=1;I=1");
    expect(nodeId).toEqual({
      namespaceIndex: 1,
      identifierType: "Numeric",
      identifier: 1,
    });
  });

  it("should parse a string node ID", () => {
    const nodeId = parseNodeID("NS=1;S=test");
    expect(nodeId).toEqual({
      namespaceIndex: 1,
      identifierType: "String",
      identifier: "test",
    });
  });

  it("should parse a GUID node ID", () => {
    const nodeId = parseNodeID("NS=1;G=test");
    expect(nodeId).toEqual({
      namespaceIndex: 1,
      identifierType: "GUID",
      identifier: "test",
    });
  });

  it("should parse a byte string node ID", () => {
    const nodeId = parseNodeID("NS=1;B=test");
    expect(nodeId).toEqual({
      namespaceIndex: 1,
      identifierType: "ByteString",
      identifier: "test",
    });
  });
  it("should return null for an invalid node ID", () => {
    const nodeId = parseNodeID("NS=1;X=test");
    expect(nodeId).toBeNull();
  });
});

describe("nodeIdToString", () => {
  it("should convert a numeric node ID to a string", () => {
    const nodeId = nodeIDToString({
      namespaceIndex: 1,
      identifierType: "Numeric",
      identifier: 1,
    });
    expect(nodeId).toEqual("NS=1;I=1");
  });

  it("should convert a string node ID to a string", () => {
    const nodeId = nodeIDToString({
      namespaceIndex: 1,
      identifierType: "String",
      identifier: "test",
    });
    expect(nodeId).toEqual("NS=1;S=test");
  });

  it("should convert a GUID node ID to a string", () => {
    const nodeId = nodeIDToString({
      namespaceIndex: 1,
      identifierType: "GUID",
      identifier: "test",
    });
    expect(nodeId).toEqual("NS=1;G=test");
  });

  it("should convert a byte string node ID to a string", () => {
    const nodeId = nodeIDToString({
      namespaceIndex: 1,
      identifierType: "ByteString",
      identifier: "test",
    });
    expect(nodeId).toEqual("NS=1;B=test");
  });
});
