// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export type Runtime = "browser" | "node";

export const detectRuntime = (): Runtime => {
  if (
    typeof process !== "undefined" &&
    process.versions != null &&
    process.versions.node != null
  )
    return "node";

  if (typeof window === "undefined" || window.document === undefined)
    console.warn("unable to safely detect runtime, assuming browser");

  return "browser";
};

export const RUNTIME = detectRuntime();
