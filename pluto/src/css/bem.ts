// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import clsx, { type ClassValue } from "clsx";

const BaseBEM = clsx;

type BaseBEMType = typeof clsx;

export interface BEM extends BaseBEMType {
  B: (...blocks: string[]) => string;
  E: (element: string) => string;
  M: (...modifiers: string[]) => string;
  BE: (block: string, ...elements: string[]) => string;
  BM: (block: string, ...modifiers: string[]) => string;
  BEM: (block: string, element: string, ...modifiers: string[]) => string;
  var: (...variables: string[]) => string;
}

const BLOCK = "-";
const ELEMENT = "__";
const MODIFIER = "--";

export const newBEM = (prefix: string): BEM => {
  // We need to define a new function to avoid reassigning the original
  // on each call to newBEM.
  const BEM_: BEM = (...args: ClassValue[]): string => BaseBEM(...args);
  BEM_.B = (...blocks) => prefix + BLOCK + blocks.join(BLOCK);
  BEM_.E = (element) => prefix + ELEMENT + element;
  BEM_.M = (...modifiers) => prefix + MODIFIER + modifiers.join("-");
  BEM_.BM = (block, ...modifiers) => BEM_.B(block) + MODIFIER + modifiers.join("-");
  BEM_.BE = (block, ...elements) => BEM_.B(block) + ELEMENT + elements.join(BLOCK);
  BEM_.BEM = (block, element, ...modifiers) =>
    BEM_.BE(block, element) + MODIFIER + modifiers.join(BLOCK);
  BEM_.var = (...variables) => MODIFIER + prefix + BLOCK + variables.join(BLOCK);
  return BEM_;
};
