// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

import { secondFactor } from "@/ui/auth/secondFactor";

type Factor = NonNullable<Parameters<typeof secondFactor.choose>[0]>[number];

const TOTP: secondFactor.CodeFactor = { strategy: "totp" };
const BACKUP: secondFactor.CodeFactor = { strategy: "backup_code" };
const EMAIL: secondFactor.CodeFactor = {
  strategy: "email_code",
  emailAddressId: "idn_email",
  safeIdentifier: "a•••@acme.com",
};
const LINK: Factor = {
  strategy: "email_link",
  emailAddressId: "idn_email",
  safeIdentifier: "a•••@acme.com",
};
const PHONE: secondFactor.CodeFactor = {
  strategy: "phone_code",
  phoneNumberId: "idn_phone",
  safeIdentifier: "+1 •••• 42",
};

type SignIn = Parameters<typeof secondFactor.prepare>[0];

describe("secondFactor", () => {
  describe("choose", () => {
    it("should prefer an authenticator app", () => {
      expect(secondFactor.choose([EMAIL, BACKUP, PHONE, TOTP])).toBe(TOTP);
    });

    it("should prefer a texted code over an emailed one", () => {
      expect(secondFactor.choose([EMAIL, PHONE])).toBe(PHONE);
    });

    it("should take the emailed code Device Trust asks for", () => {
      expect(secondFactor.choose([EMAIL])).toBe(EMAIL);
    });

    it("should choose nothing when Clerk offers only backup codes or no factors", () => {
      expect(secondFactor.choose([BACKUP])).toBeNull();
      expect(secondFactor.choose(null)).toBeNull();
      expect(secondFactor.choose([LINK])).toBeNull();
    });
  });

  describe("backupCode", () => {
    it("should find the backup code factor", () => {
      expect(secondFactor.backupCode([TOTP, BACKUP])).toBe(BACKUP);
      expect(secondFactor.backupCode([TOTP])).toBeNull();
    });
  });

  describe("prompt", () => {
    it("should say where each code comes from", () => {
      expect(secondFactor.prompt(TOTP)).toBe(
        "Enter the code from your authenticator app",
      );
      expect(secondFactor.prompt(BACKUP)).toBe("Enter one of your backup codes");
      expect(secondFactor.prompt(EMAIL)).toBe(
        "Enter the code we sent to a•••@acme.com",
      );
      expect(secondFactor.prompt(PHONE)).toBe("Enter the code we sent to +1 •••• 42");
    });
  });

  describe("prepare", () => {
    const signIn = (): { resource: SignIn; send: ReturnType<typeof vi.fn> } => {
      const send = vi.fn(async () => ({}));
      return { resource: { prepareSecondFactor: send } as unknown as SignIn, send };
    };

    it("should send an emailed code to the factor's address", async () => {
      const { resource, send } = signIn();
      await secondFactor.prepare(resource, EMAIL);
      expect(send).toHaveBeenCalledWith({
        strategy: "email_code",
        emailAddressId: "idn_email",
      });
    });

    it("should send a texted code to the factor's number", async () => {
      const { resource, send } = signIn();
      await secondFactor.prepare(resource, PHONE);
      expect(send).toHaveBeenCalledWith({
        strategy: "phone_code",
        phoneNumberId: "idn_phone",
      });
    });

    it("should send nothing for an authenticator app or a backup code", async () => {
      const { resource, send } = signIn();
      await secondFactor.prepare(resource, TOTP);
      await secondFactor.prepare(resource, BACKUP);
      expect(send).not.toHaveBeenCalled();
    });
  });
});
