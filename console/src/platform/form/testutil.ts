// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * Returns the rendered KeyValueEditor rows in document order. Rows carry no
 * accessible handle of their own, so the structural class selector lives here.
 */
export const getKVRows = (container: HTMLElement): HTMLElement[] =>
  Array.from(container.querySelectorAll<HTMLElement>(".console-kv-row"));
