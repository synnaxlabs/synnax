// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ArcServices } from "@/arc/services";
import { Hardware } from "@/hardware";
import { type Import } from "@/import";
import { LinePlotServices } from "@/lineplot/services";
import { LogServices } from "@/log/services";
import { SchematicServices } from "@/schematic/services";
import { TableServices } from "@/table/services";

export const FILE_INGESTERS: Import.FileIngesters = {
  ...ArcServices.FILE_INGESTERS,
  ...Hardware.FILE_INGESTERS,
  ...LinePlotServices.FILE_INGESTERS,
  ...LogServices.FILE_INGESTERS,
  ...SchematicServices.FILE_INGESTERS,
  ...TableServices.FILE_INGESTERS,
};
