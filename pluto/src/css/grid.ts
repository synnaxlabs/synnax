// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CSSProperties } from "react";

interface CSSGridEntry {
  startLabel: string;
  endLabel: string;
  size: number | string;
}

export class CSSGridBuilder {
  rows: CSSGridEntry[] = [];
  columns: CSSGridEntry[] = [];

  addRow(start: string, end: string, size: number | string): this {
    this.rows.push({ startLabel: start, endLabel: end, size });
    return this;
  }

  addColumn(start: string, end: string, size: number | string): this {
    this.columns.push({ startLabel: start, endLabel: end, size });
    return this;
  }

  build(): CSSProperties {
    return {
      display: "grid",
      gridTemplateRows: this.rows
        .map((r, i) => {
          let t = i === 0 ? "[" : "";
          t += `${r.startLabel}] ${r.size}${typeof r.size === "number" ? "px" : ""} [${
            r.endLabel
          }`;
          t += i === this.rows.length - 1 ? "]" : "";
          return t;
        })
        .join(" "),
      gridTemplateColumns: this.columns
        .map((c, i) => {
          let t = i === 0 ? "[" : "";
          t += `${c.startLabel}] ${c.size}${typeof c.size === "number" ? "px" : ""} [${
            c.endLabel
          }`;
          t += i === this.columns.length - 1 ? "]" : "";
          return t;
        })
        .join(" "),
    };
  }
}
