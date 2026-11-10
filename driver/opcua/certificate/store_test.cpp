// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <filesystem>

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/opcua/certificate/store.h"

namespace driver::opcua::certificate {
namespace {
std::size_t count_files(const std::filesystem::path &dir) {
    if (!std::filesystem::exists(dir)) return 0;
    std::size_t n = 0;
    for ([[maybe_unused]] const auto &entry: std::filesystem::directory_iterator(dir))
        n++;
    return n;
}

class StoreTest : public ::testing::Test {
protected:
    std::filesystem::path dir;

    void SetUp() override {
        this->dir = std::filesystem::temp_directory_path() /
                    ("opcua-store-" + x::uuid::create().to_string()) / "pki";
    }

    void TearDown() override { std::filesystem::remove_all(this->dir.parent_path()); }

    [[nodiscard]] std::size_t cert_count() const {
        return count_files(this->dir / "own" / "certs");
    }

    [[nodiscard]] std::size_t key_count() const {
        return count_files(this->dir / "own" / "private");
    }
};
}

/// @brief it should make an identity on first use and keep it on disk.
TEST_F(StoreTest, CurrentCreatesOnFirstUse) {
    Store store(StoreConfig{.dir = this->dir, .host = "test-host"});
    const auto id = ASSERT_NIL_P(store.current());
    ASSERT_TRUE(id.certificate.application_uri().starts_with("urn:synnax:driver:"));
    ASSERT_TRUE(id.certificate.subject().starts_with("CN=Synnax Driver "));
    ASSERT_EQ(this->cert_count(), 1);
    ASSERT_EQ(this->key_count(), 1);
    const auto again = ASSERT_NIL_P(store.current());
    ASSERT_EQ(again.certificate, id.certificate);
}

/// @brief it should load the same identity in a new Store over the same directory.
TEST_F(StoreTest, CurrentPersists) {
    Store first(StoreConfig{.dir = this->dir});
    const auto id = ASSERT_NIL_P(first.current());
    Store second(StoreConfig{.dir = this->dir});
    const auto loaded = ASSERT_NIL_P(second.current());
    ASSERT_EQ(loaded.certificate, id.certificate);
    ASSERT_EQ(loaded.private_key, id.private_key);
}

#ifndef _WIN32
/// @brief it should let only the owner read the private key.
TEST_F(StoreTest, KeyPermissions) {
    Store store(StoreConfig{.dir = this->dir});
    ASSERT_NIL_P(store.current());
    using std::filesystem::perms;
    const auto private_dir = this->dir / "own" / "private";
    ASSERT_EQ(std::filesystem::status(private_dir).permissions(), perms::owner_all);
    for (const auto &entry: std::filesystem::directory_iterator(private_dir))
        ASSERT_EQ(
            std::filesystem::status(entry).permissions(),
            perms::owner_read | perms::owner_write
        );
}
#endif

/// @brief it should not start renewal outside the renewal window.
TEST_F(StoreTest, RenewOutsideWindow) {
    Store store(StoreConfig{.dir = this->dir});
    ASSERT_NIL_P(store.current());
    ASSERT_NIL(store.renew());
    const auto next = ASSERT_NIL_P(store.next());
    ASSERT_FALSE(next.has_value());
    ASSERT_EQ(this->cert_count(), 1);
}

/// @brief it should issue the next identity inside the renewal window and keep using
/// the current one.
TEST_F(StoreTest, RenewInsideWindow) {
    x::telem::TimeStamp now = x::telem::TimeStamp::now();
    Store store(StoreConfig{.dir = this->dir, .now = [&] { return now; }});
    const auto current = ASSERT_NIL_P(store.current());
    now = current.certificate.not_after() - 30 * x::telem::DAY;
    ASSERT_NIL(store.renew());
    const auto next = ASSERT_NIL_P(store.next());
    ASSERT_TRUE(next.has_value());
    ASSERT_NE(next.value().certificate, current.certificate);
    ASSERT_EQ(
        next.value().certificate.application_uri(),
        current.certificate.application_uri()
    );
    const auto still = ASSERT_NIL_P(store.current());
    ASSERT_EQ(still.certificate, current.certificate);
    ASSERT_NIL(store.renew());
    ASSERT_EQ(this->cert_count(), 2);
}

/// @brief it should switch to the next identity on promote and delete the old one.
TEST_F(StoreTest, Promote) {
    x::telem::TimeStamp now = x::telem::TimeStamp::now();
    Store store(StoreConfig{.dir = this->dir, .now = [&] { return now; }});
    const auto old = ASSERT_NIL_P(store.current());
    now = old.certificate.not_after() - 30 * x::telem::DAY;
    ASSERT_NIL(store.renew());
    const auto next = ASSERT_NIL_P(store.next());
    ASSERT_NIL(store.promote());
    const auto current = ASSERT_NIL_P(store.current());
    ASSERT_EQ(current.certificate, next.value().certificate);
    const auto none = ASSERT_NIL_P(store.next());
    ASSERT_FALSE(none.has_value());
    ASSERT_EQ(this->cert_count(), 1);
    ASSERT_EQ(this->key_count(), 1);
    Store reloaded(StoreConfig{.dir = this->dir, .now = [&] { return now; }});
    const auto loaded = ASSERT_NIL_P(reloaded.current());
    ASSERT_EQ(loaded.certificate, next.value().certificate);
}

/// @brief it should switch to the next identity when the current one expires.
TEST_F(StoreTest, SwitchAtExpiry) {
    const auto start = x::telem::TimeStamp::now();
    Store short_lived(StoreConfig{.dir = this->dir, .validity = 10 * x::telem::DAY});
    const auto old = ASSERT_NIL_P(short_lived.current());
    x::telem::TimeStamp now = start + 5 * x::telem::DAY;
    Store store(
        StoreConfig{
            .dir = this->dir,
            .validity = 100 * x::telem::DAY,
            .now = [&] { return now; },
        }
    );
    ASSERT_NIL(store.renew());
    const auto next = ASSERT_NIL_P(store.next());
    ASSERT_TRUE(next.has_value());
    const auto before = ASSERT_NIL_P(store.current());
    ASSERT_EQ(before.certificate, old.certificate);
    now = old.certificate.not_after() + x::telem::DAY;
    const auto after = ASSERT_NIL_P(store.current());
    ASSERT_EQ(after.certificate, next.value().certificate);
    ASSERT_EQ(this->cert_count(), 1);
}

/// @brief it should replace an expired identity when no next one exists.
TEST_F(StoreTest, ReplaceExpired) {
    x::telem::TimeStamp now = x::telem::TimeStamp::now();
    Store store(StoreConfig{.dir = this->dir, .now = [&] { return now; }});
    const auto old = ASSERT_NIL_P(store.current());
    now = old.certificate.not_after() + x::telem::DAY;
    const auto replaced = ASSERT_NIL_P(store.current());
    ASSERT_NE(replaced.certificate, old.certificate);
    ASSERT_EQ(
        replaced.certificate.application_uri(),
        old.certificate.application_uri()
    );
    ASSERT_EQ(this->cert_count(), 1);
}

/// @brief it should replace the current and next identities on regenerate.
TEST_F(StoreTest, Regenerate) {
    x::telem::TimeStamp now = x::telem::TimeStamp::now();
    Store store(StoreConfig{.dir = this->dir, .now = [&] { return now; }});
    const auto old = ASSERT_NIL_P(store.current());
    now = old.certificate.not_after() - 30 * x::telem::DAY;
    ASSERT_NIL(store.renew());
    const auto next = ASSERT_NIL_P(store.next());
    const auto fresh = ASSERT_NIL_P(store.regenerate());
    ASSERT_NE(fresh.certificate, old.certificate);
    ASSERT_NE(fresh.certificate, next.value().certificate);
    ASSERT_EQ(fresh.certificate.application_uri(), old.certificate.application_uri());
    const auto current = ASSERT_NIL_P(store.current());
    ASSERT_EQ(current.certificate, fresh.certificate);
    const auto none = ASSERT_NIL_P(store.next());
    ASSERT_FALSE(none.has_value());
    ASSERT_EQ(this->cert_count(), 1);
    ASSERT_EQ(this->key_count(), 1);
}
}
