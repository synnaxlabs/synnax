// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Triggers } from "@/triggers";

describe("Triggers", () => {
  describe("Triggers.filter", () => {
    describe("not loose", () => {
      it("Should return an empty list when no triggers match", () => {
        expect(
          Triggers.filter(
            [
              ["A", "B"],
              ["A", "C"],
            ],
            [["A", "D"]],
          ),
        ).toEqual([]);
      });
      it("Should return a list of triggers that match", () => {
        expect(
          Triggers.filter(
            [
              ["A", "B"],
              ["A", "C"],
            ],
            [["A", "B"]],
          ),
        ).toEqual([["A", "B"]]);
      });
      it("Should not match loose triggers", () => {
        expect(Triggers.filter([["A"], ["A", "C"]], [["A", "B"]])).toEqual([]);
      });
      it("Should match multiple triggers", () => {
        expect(
          Triggers.filter(
            [
              ["A", "B"],
              ["A", "C"],
            ],
            [
              ["A", "B"],
              ["A", "C"],
            ],
          ),
        ).toEqual([
          ["A", "B"],
          ["A", "C"],
        ]);
      });
    });
    describe("loose", () => {
      it("Should return an empty list when no triggers match", () => {
        expect(
          Triggers.filter(
            [
              ["A", "B"],
              ["A", "C"],
            ],
            [["A", "D"]],
            { loose: true },
          ),
        ).toEqual([]);
      });
      it("Should return a list of triggers that match", () => {
        expect(
          Triggers.filter([["A"], ["A", "C"]], [["A", "B"]], { loose: true }),
        ).toEqual([["A"]]);
      });
      it("should return an empty list when no triggers match", () => {
        expect(
          Triggers.filter(
            [
              ["A", "B"],
              ["A", "C"],
            ],
            [["A"]],
            { loose: true },
          ),
        ).toEqual([]);
      });
    });
    describe("Triggers.purge", () => {
      it("Should correctly removed triggers from a list", () => {
        expect(
          Triggers.purge(
            [
              ["A", "B"],
              ["A", "C"],
            ],
            [["A", "B"]],
          ),
        ).toEqual([["A", "C"]]);
      });
    });
    describe("Diff", () => {
      it("Should correctly diff two lists of triggers", () => {
        expect(
          Triggers.diff(
            [
              ["A", "B"],
              ["A", "C"],
              ["A", "E"],
            ],
            [
              ["A", "B"],
              ["A", "C"],
              ["A", "D"],
            ],
          ),
        ).toEqual([[["A", "E"]], [["A", "D"]]]);
      });
    });
    describe("match", () => {
      it("should match the trigger correctly", () => {
        expect(Triggers.match);
      });
    });
    describe("Config", () => {
      describe("determineMode", () => {
        it("should select the matching mode with the highest complexity", () => {
          const config: Triggers.ModeConfig<"a" | "b"> = {
            defaultMode: "a",
            a: [["Shift"]],
            b: [["Shift", "Control"]],
          };
          expect(Triggers.determineMode(config, [["Shift", "Control"]])).toEqual("b");
          expect(Triggers.determineMode(config, [["Shift"]])).toEqual("a");
        });
        it("should correctly match loose trigers", () => {
          const config: Triggers.ModeConfig<"a" | "b"> = {
            defaultMode: "a",
            a: [["Shift"]],
            b: [["Shift", "Control"]],
          };
          expect(
            Triggers.determineMode(config, [["Shift", "Control"]], { loose: true }),
          ).toEqual("b");
          expect(Triggers.determineMode(config, [["Shift"]], { loose: true })).toEqual(
            "a",
          );
        });
      });
    });
  });
});
