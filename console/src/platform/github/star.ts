// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { invoke } from "@tauri-apps/api/core";
import z from "zod";

/** The repository page, where a user without the GitHub CLI stars Synnax. */
export const URL = "https://github.com/synnaxlabs/synnax";

const statusZ = z.enum(["starred", "not_starred", "unknown"]);
type Status = z.infer<typeof statusZ>;

/**
 * Asks the GitHub CLI whether the user starred Synnax. Unknown when the CLI is missing
 * or signed out.
 */
export const retrieveStatus = async (): Promise<Status> =>
  statusZ.parse(await invoke("github_star_status"));

/** @returns true when the GitHub CLI starred Synnax. */
export const star = async (): Promise<boolean> =>
  z.boolean().parse(await invoke("github_star"));
