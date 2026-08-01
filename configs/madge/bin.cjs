#!/usr/bin/env node
// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// madge needs the TypeScript JS Compiler API, which the native TS 7 package no longer
// ships. This package depends on TypeScript 6 so madge and its detectives resolve a
// typescript peer that still provides the API.
require("madge/bin/cli.js");
