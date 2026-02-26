// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Input, Tabs, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useState } from "react";
import { FaGithub } from "react-icons/fa";

import { pb } from "@/util/pocketbase";

export interface AuthModalProps {
  visible: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

type Tab = "signin" | "signup";

export const AuthModal = ({
  visible,
  onClose,
  onSuccess,
}: AuthModalProps): ReactElement | null => {
  const [tab, setTab] = useState<Tab>("signin");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [name, setName] = useState("");
  const [company, setCompany] = useState("");

  const resetForm = useCallback(() => {
    setEmail("");
    setPassword("");
    setName("");
    setCompany("");
    setError(null);
  }, []);

  const handleSignIn = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await pb.collection("users").authWithPassword(email, password);
      onSuccess();
      onClose();
      resetForm();
    } catch {
      setError("Invalid email or password.");
    } finally {
      setLoading(false);
    }
  }, [email, password, onSuccess, onClose, resetForm]);

  const handleSignUp = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await pb.collection("users").create({
        email,
        password,
        passwordConfirm: password,
        name,
        company,
      });
      await pb.collection("users").authWithPassword(email, password);
      onSuccess();
      onClose();
      resetForm();
    } catch {
      setError("Failed to create account. The email may already be in use.");
    } finally {
      setLoading(false);
    }
  }, [email, password, name, company, onSuccess, onClose, resetForm]);

  const handleGitHub = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await pb.collection("users").authWithOAuth2({ provider: "github" });
      onSuccess();
      onClose();
      resetForm();
    } catch {
      setError("GitHub authentication failed.");
    } finally {
      setLoading(false);
    }
  }, [onSuccess, onClose, resetForm]);

  const handleTabSelect = useCallback(
    (key: string) => {
      setTab(key as Tab);
      setError(null);
    },
    [],
  );

  if (!visible) return null;

  return (
    <div className="auth-modal-overlay" onClick={onClose}>
      <div className="auth-modal" onClick={(e) => e.stopPropagation()}>
        <div className="auth-modal__header">
          <Text.Text level="h4">Sign in to download</Text.Text>
          <Button.Button variant="text" size="small" onClick={onClose}>
            &times;
          </Button.Button>
        </div>

        <Tabs.Tabs
          tabs={[
            { tabKey: "signin", name: "Sign In" },
            { tabKey: "signup", name: "Sign Up" },
          ]}
          selected={tab}
          onSelect={handleTabSelect}
        />

        <div className="auth-modal__body">
          {tab === "signup" && (
            <>
              <label className="auth-modal__label">
                <Text.Text level="small">Name</Text.Text>
                <Input.Text value={name} onChange={setName} />
              </label>
              <label className="auth-modal__label">
                <Text.Text level="small">Company (optional)</Text.Text>
                <Input.Text value={company} onChange={setCompany} />
              </label>
            </>
          )}

          <label className="auth-modal__label">
            <Text.Text level="small">Email</Text.Text>
            <Input.Text value={email} onChange={setEmail} />
          </label>

          <label className="auth-modal__label">
            <Text.Text level="small">Password</Text.Text>
            <Input.Text value={password} onChange={setPassword} type="password" />
          </label>

          {error != null && (
            <Text.Text level="small" className="auth-modal__error">
              {error}
            </Text.Text>
          )}

          <Button.Button
            variant="filled"
            disabled={loading}
            onClick={tab === "signin" ? handleSignIn : handleSignUp}
          >
            {tab === "signin" ? "Sign In" : "Create Account"}
          </Button.Button>

          <div className="auth-modal__divider">
            <Text.Text level="small">or</Text.Text>
          </div>

          <Button.Button variant="outlined" disabled={loading} onClick={handleGitHub}>
            <FaGithub />
            Continue with GitHub
          </Button.Button>
        </div>
      </div>
    </div>
  );
};
