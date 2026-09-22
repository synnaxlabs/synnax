// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

const SITE_URL = "https://docs.synnaxlabs.com";

/** The page that issues a token for a host fingerprint. */
export const ACTIVATE_URL = `${SITE_URL}/account/licenses/activate`;

/** The page that lists the account's licenses and linked machines. */
export const ACCOUNT_URL = `${SITE_URL}/account`;

/** The page that links a Synnax Desktop machine to an account. */
export const SIGN_IN_URL = `${SITE_URL}/desktop/sign-in`;

/** The route that renews a linked machine's license. */
export const RENEW_URL = `${SITE_URL}/api/desktop/renew`;
