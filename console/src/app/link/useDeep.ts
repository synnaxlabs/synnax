// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Core } from "@/feature/core";
import { Link } from "@/feature/link";
import { Link as PLink } from "@/platform/link";
import { Session } from "@/session";

const useLinks = (): void => Link.useDeep(Core.useLink(), PLink.useHandlers());

// Synnax Desktop takes one link, its login, and Account.Guard handles that one.
export const useDeep: () => void = Session.Runtime.LINKS_DISABLED ? () => {} : useLinks;
