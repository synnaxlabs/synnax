// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Clerk } from "@/ui/clerk";

type SignInResource = NonNullable<Clerk["client"]>["signIn"];
type Factor = NonNullable<SignInResource["supportedSecondFactors"]>[number];

/** CodeFactor is a second step the user finishes by typing a code. */
export type CodeFactor = Extract<
  Factor,
  { strategy: "totp" | "phone_code" | "email_code" | "backup_code" }
>;

const PREFERENCE: CodeFactor["strategy"][] = ["totp", "phone_code", "email_code"];

/**
 * choose picks the second step to ask for: an authenticator app first, then a texted
 * or emailed code. Returns null when Clerk offers none of them.
 */
export const choose = (factors: Factor[] | null): CodeFactor | null => {
  for (const strategy of PREFERENCE) {
    const found = factors?.find((f): f is CodeFactor => f.strategy === strategy);
    if (found != null) return found;
  }
  return null;
};

/** backupCode returns the backup code factor when Clerk offers one. */
export const backupCode = (factors: Factor[] | null): CodeFactor | null =>
  factors?.find((f): f is CodeFactor => f.strategy === "backup_code") ?? null;

/** prompt tells the user where to find the code for `factor`. */
export const prompt = (factor: CodeFactor): string => {
  switch (factor.strategy) {
    case "totp":
      return "Enter the code from your authenticator app";
    case "backup_code":
      return "Enter one of your backup codes";
    case "phone_code":
    case "email_code":
      return `Enter the code we sent to ${factor.safeIdentifier}`;
  }
};

/** prepare asks Clerk to send the code for a texted or emailed factor. */
export const prepare = async (
  signIn: SignInResource,
  factor: CodeFactor,
): Promise<void> => {
  if (factor.strategy === "phone_code")
    await signIn.prepareSecondFactor({
      strategy: "phone_code",
      phoneNumberId: factor.phoneNumberId,
    });
  else if (factor.strategy === "email_code")
    await signIn.prepareSecondFactor({
      strategy: "email_code",
      emailAddressId: factor.emailAddressId,
    });
};
