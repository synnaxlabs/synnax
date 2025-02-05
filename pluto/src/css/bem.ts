// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import clsx, { type ClassValue } from "clsx";
import { type CSSProperties } from "react";

const CoreBEM = clsx;

type CoreBEMType = typeof clsx;

export interface BEM extends CoreBEMType {
  B: (block: string) => string;
  E: (element: string) => string;
  M: (modifier: string) => string;
  BE: (block: string, element: string) => string;
  BM: (block: string, modifier: string) => string;
  BEM: (block: string, element: string, modifier: string) => string;
  extend: (prefix: string) => BEM;
  var: (variable: string) => string;
}

const BLOCK = "-";
const ELEMENT = "__";
const MODIFIER = "--";

export const newBEM = (prefix: string): BEM => {
  // We need to define a new function to avoid reassigning the original
  // on each call to newBEM.
  const BEM_: BEM = (...args: ClassValue[]): string => CoreBEM(...args);
  BEM_.B = (block) => prefix + BLOCK + block;
  BEM_.E = (element) => prefix + ELEMENT + element;
  BEM_.M = (modifier) => prefix + MODIFIER + modifier;
  BEM_.BM = (block, modifier) => BEM_.B(block) + MODIFIER + modifier;
  BEM_.BE = (block, element) => BEM_.B(block) + ELEMENT + element;
  BEM_.BEM = (block, element, modifier) =>
    BEM_.BE(block, element) + MODIFIER + modifier;
  BEM_.extend = (prefix_) => newBEM(BEM_.B(prefix_));
  BEM_.var = (variable) =>
    (MODIFIER + prefix + BLOCK + variable) as keyof CSSProperties;
  return BEM_;
};
