// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

import { Import } from "@/import";
import { create } from "@/lineplot/LinePlot";
import { anyStateZ } from "@/lineplot/slice";

export const ingest: Import.Ingestor = ({ data, name, key, layout }) => {
  const state = anyStateZ.parse(JSON.parse(data));
  // create with an undefined key so we do not have to worry about existing line plots
  return create({ ...state, name, key, ...layout });
};

export const import_ = Import.createImporter(ingest, "line plot");

export const useImport = (workspaceKey?: string) =>
  Import.useImport(import_, workspaceKey);
