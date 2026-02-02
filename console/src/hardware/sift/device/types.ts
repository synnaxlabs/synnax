// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export interface Properties {
  [key: string]: unknown;
  uri: string;
  api_key: string;
}

export const ZERO_PROPERTIES: Properties = {
  uri: "",
  api_key: "",
};

export type Make = "sift";
export type Model = "cloud";

export const MAKE: Make = "sift";
export const MODEL: Model = "cloud";
