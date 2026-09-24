// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// <reference types="vite/client" />

declare const IS_DEV: boolean;
/**
 * True in a Synnax Desktop build. Only the composition root (`app/`) reads it; lower
 * layers stay the same for both apps.
 */
declare const DESKTOP: boolean;

interface MonacoEnvironment {
  getWorker: (moduleId: string, label: string) => Worker;
}

declare global {
  interface Window {
    MonacoEnvironment: MonacoEnvironment;
  }
}
