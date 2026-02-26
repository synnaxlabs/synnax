// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  createContext,
  type ReactElement,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";

import { pb } from "@/util/pocketbase";

export interface User {
  email: string;
  name: string;
}

export interface AuthContextValue {
  isAuthenticated: boolean;
  user: User | null;
  showAuthModal: () => void;
  hideAuthModal: () => void;
  modalVisible: boolean;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue>({
  isAuthenticated: false,
  user: null,
  showAuthModal: () => {},
  hideAuthModal: () => {},
  modalVisible: false,
  logout: () => {},
});

export const useAuth = (): AuthContextValue => useContext(AuthContext);

export interface AuthProviderProps {
  children: ReactNode;
}

export const AuthProvider = ({ children }: AuthProviderProps): ReactElement => {
  const [isAuthenticated, setIsAuthenticated] = useState(pb.authStore.isValid);
  const [user, setUser] = useState<User | null>(() => {
    const record = pb.authStore.record;
    if (record == null) return null;
    return { email: record.email, name: record.name ?? "" };
  });
  const [modalVisible, setModalVisible] = useState(false);

  useEffect(() => {
    const unsubscribe = pb.authStore.onChange(() => {
      setIsAuthenticated(pb.authStore.isValid);
      const record = pb.authStore.record;
      if (record != null) {
        setUser({ email: record.email, name: record.name ?? "" });
      } else {
        setUser(null);
      }
    });
    return unsubscribe;
  }, []);

  const showAuthModal = useCallback(() => setModalVisible(true), []);
  const hideAuthModal = useCallback(() => setModalVisible(false), []);

  const logout = useCallback(() => {
    pb.authStore.clear();
  }, []);

  const value = useMemo(
    () => ({
      isAuthenticated,
      user,
      showAuthModal,
      hideAuthModal,
      modalVisible,
      logout,
    }),
    [isAuthenticated, user, showAuthModal, hideAuthModal, modalVisible, logout],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};
