// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, type ReactNode, useCallback } from "react";

import { AuthModal } from "@/components/auth/AuthModal";
import { AuthProvider, useAuth } from "@/components/auth/AuthProvider";

export interface AuthGateProps {
  children: ReactNode;
}

const AuthGateInner = ({ children }: AuthGateProps): ReactElement => {
  const { modalVisible, hideAuthModal } = useAuth();
  const handleSuccess = useCallback(() => {}, []);
  return (
    <>
      {children}
      <AuthModal
        visible={modalVisible}
        onClose={hideAuthModal}
        onSuccess={handleSuccess}
      />
    </>
  );
};

export const AuthGate = ({ children }: AuthGateProps): ReactElement => (
  <AuthProvider>
    <AuthGateInner>{children}</AuthGateInner>
  </AuthProvider>
);
