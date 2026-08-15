// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Panel } from "@/feature/panel";
import { Project } from "@/feature/project";
import { Import } from "@/platform/import";
import { Selector } from "@/platform/selector";

export const Mosaic = (): ReactElement => {
  const handleFileDrop = Import.useFileDrop({ ingestDirectory: Project.ingest });
  return (
    <Panel.Mosaic onCreateTab={Selector.createEmptyTab} onFileDrop={handleFileDrop} />
  );
};
