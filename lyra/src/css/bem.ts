// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export interface BEM {
  /** @returns the class name for a block. */
  B: (...blocks: string[]) => string;
  /** @returns the class name for an element of the enclosing block. */
  E: (element: string) => string;
  /** @returns the class name for a modifier. */
  M: (...modifiers: string[]) => string;
  /** @returns the class name for an element of the given block. */
  BE: (block: string, ...elements: string[]) => string;
  /** @returns the class name for a modifier of the given block. */
  BM: (block: string, ...modifiers: string[]) => string;
  /** @returns the class name for a modified element of the given block. */
  BEM: (block: string, element: string, ...modifiers: string[]) => string;
  /** @returns the name of a custom property, including the leading dashes. */
  variable: (...variables: string[]) => string;
}

const BLOCK = "-";
const ELEMENT = "__";
const MODIFIER = "--";

export const newBEM = (prefix: string): BEM => {
  const B: BEM["B"] = (...blocks) => prefix + BLOCK + blocks.join(BLOCK);
  const E: BEM["E"] = (element) => prefix + ELEMENT + element;
  const M: BEM["M"] = (...modifiers) => prefix + MODIFIER + modifiers.join("-");
  const BM: BEM["BM"] = (block, ...modifiers) =>
    B(block) + MODIFIER + modifiers.join("-");
  const BE: BEM["BE"] = (block, ...elements) =>
    B(block) + ELEMENT + elements.join(BLOCK);
  const BEM: BEM["BEM"] = (block, element, ...modifiers) =>
    BE(block, element) + MODIFIER + modifiers.join(BLOCK);
  const variable: BEM["variable"] = (...variables) =>
    MODIFIER + prefix + BLOCK + variables.join(BLOCK);
  return { B, E, M, BM, BE, BEM, variable };
};
