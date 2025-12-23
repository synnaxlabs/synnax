// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  ResourceAnalyzer,
  type ResourceContext,
} from "@/perf/analyzer/resource-analyzer";
import { type GpuReport } from "@/perf/analyzer/types";
import { THRESHOLDS } from "@/perf/constants";

export type GpuContext = ResourceContext;

export class GpuAnalyzer extends ResourceAnalyzer {
  constructor() {
    super(THRESHOLDS.gpu, THRESHOLDS.gpuAvg);
  }

  override analyze(ctx: GpuContext): GpuReport {
    return super.analyze(ctx) as GpuReport;
  }
}
