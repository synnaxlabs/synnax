// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { sanitizeFileName } from "@/strings/sanitizeFileName";

describe("sanitizeFileName", () => {
  it("returns ordinary names unchanged", () => {
    expect(sanitizeFileName("My Project")).toEqual("My Project");
    expect(sanitizeFileName("metrics_2025")).toEqual("metrics_2025");
    expect(sanitizeFileName("name-with-dashes")).toEqual("name-with-dashes");
    expect(sanitizeFileName("file.json")).toEqual("file.json");
  });

  it("replaces forward and back slashes", () => {
    expect(sanitizeFileName("a/b")).toEqual("a_b");
    expect(sanitizeFileName("a\\b")).toEqual("a_b");
    expect(sanitizeFileName("path/to\\file")).toEqual("path_to_file");
  });

  it("replaces Windows-reserved characters", () => {
    expect(sanitizeFileName('a<b>c:d"e|f?g*h')).toEqual("a_b_c_d_e_f_g_h");
  });

  it("replaces control characters", () => {
    expect(sanitizeFileName("a\x00b\x1fc\x7f")).toEqual("a_b_c_");
  });

  it("drops trailing dots and spaces", () => {
    expect(sanitizeFileName("report.")).toEqual("report");
    expect(sanitizeFileName("report ")).toEqual("report");
    expect(sanitizeFileName("report. . ")).toEqual("report");
  });

  it("prefixes an underscore to a Windows device name", () => {
    expect(sanitizeFileName("CON")).toEqual("_CON");
    expect(sanitizeFileName("nul")).toEqual("_nul");
    expect(sanitizeFileName("COM1")).toEqual("_COM1");
    expect(sanitizeFileName("lpt9")).toEqual("_lpt9");
    expect(sanitizeFileName("aux.tar")).toEqual("_aux.tar");
  });

  it("leaves a name that only starts with a device name alone", () => {
    expect(sanitizeFileName("console")).toEqual("console");
    expect(sanitizeFileName("com10")).toEqual("com10");
  });

  it("returns an empty string unchanged", () => {
    expect(sanitizeFileName("")).toEqual("");
  });

  it("returns an empty string for a name of dots and spaces alone", () => {
    expect(sanitizeFileName(". . ")).toEqual("");
  });

  it("does not collapse repeated unsafe characters", () => {
    expect(sanitizeFileName("a///b")).toEqual("a___b");
  });
});
