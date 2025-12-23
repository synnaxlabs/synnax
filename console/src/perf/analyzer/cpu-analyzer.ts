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
import { type CpuReport } from "@/perf/analyzer/types";
import { THRESHOLDS } from "@/perf/constants";

export type CpuContext = ResourceContext;

export class CpuAnalyzer extends ResourceAnalyzer {
  constructor() {
    super(THRESHOLDS.cpu, THRESHOLDS.cpuAvg);
  }

  override analyze(ctx: CpuContext): CpuReport {
    return super.analyze(ctx) as CpuReport;
  }
}
