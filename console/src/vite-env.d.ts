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
/** The port of the Core that a dev Console connects to. */
declare const CORE_PORT: number;
/**
 * True in a Synnax Desktop build. Only the composition root (`app/`) reads it; lower
 * layers stay the same for both apps.
 */
declare const DESKTOP: boolean;
/** True only in a Desktop release that CI builds. Local and test builds leave it off. */
declare const RELEASE: boolean;

interface MonacoEnvironment {
  getWorker: (moduleId: string, label: string) => Worker;
}

declare global {
  interface Window {
    MonacoEnvironment: MonacoEnvironment;
  }
}
