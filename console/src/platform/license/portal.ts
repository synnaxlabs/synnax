// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

const PORTAL_URL = "https://docs.synnaxlabs.com";

/** The portal page that issues a token for a host fingerprint. */
export const PORTAL_ACTIVATE_URL = `${PORTAL_URL}/portal/licenses/activate`;

/** The portal page that lists the account's licenses and linked machines. */
export const PORTAL_LICENSES_URL = `${PORTAL_URL}/portal`;

/** The portal page that links a Synnax Desktop machine to an account. */
export const PORTAL_SIGN_IN_URL = `${PORTAL_URL}/desktop/sign-in`;

/** The portal route that renews a linked machine's license. */
export const PORTAL_RENEW_URL = `${PORTAL_URL}/api/portal/desktop/renew`;
