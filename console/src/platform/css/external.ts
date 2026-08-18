// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CSS as PCSS } from "@synnaxlabs/pluto";

const bem = PCSS.newBEM("console");

/** Joins class values into a single class name, dropping the falsy ones. */
export const cx = PCSS.cx;

/** @returns the class name for a block. */
export const B = bem.B;
/** @returns the class name for an element of the enclosing block. */
export const E = bem.E;
/** @returns the class name for a modifier. */
export const M = bem.M;
/** @returns the class name for an element of the given block. */
export const BE = bem.BE;
/** @returns the class name for a modifier of the given block. */
export const BM = bem.BM;
/** @returns the class name for a modified element of the given block. */
export const BEM = bem.BEM;

const cssVar = bem.var;
/** @returns the name of a custom property, including the leading dashes. */
export { cssVar as var };

export type VarProperties = PCSS.VarProperties;
