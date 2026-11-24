// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <filesystem>
#include <memory>
#include <string>
#include <variant>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/opcua/certificate/records.h"
#include "driver/opcua/certificate/store.h"

namespace driver::opcua::certificate {
namespace {
using synnax::certificate::FileCertificate;

class RecordsTest : public ::testing::Test {
protected:
    synnax::Synnax client = new_test_client();
    synnax::rack::Rack rack;
    std::shared_ptr<synnax::Synnax> as_rack;
    std::filesystem::path dir;
    x::telem::TimeStamp now = x::telem::TimeStamp::now();
    std::shared_ptr<Store> store;

    void SetUp() override {
        this->rack = ASSERT_NIL_P(this->client.racks.create("opc_records_test_rack"));
        this->as_rack = std::make_shared<synnax::Synnax>(
            rack_client_config(this->client, this->rack.key)
        );
        this->dir = std::filesystem::temp_directory_path() /
                    ("opcua-records-" + x::uuid::create().to_string());
        this->store = std::make_shared<Store>(
            StoreConfig{.dir = this->dir, .now = [this] { return this->now; }}
        );
    }

    void TearDown() override { std::filesystem::remove_all(this->dir); }

    [[nodiscard]] Records records() const {
        return Records(this->store, this->as_rack, this->rack.key);
    }

    /// @returns the file certificates on the rack.
    [[nodiscard]] std::vector<FileCertificate> on_rack() const {
        const auto all = ASSERT_NIL_P(this->client.certificates.retrieve(
            synnax::certificate::Where{.racks = this->rack.key}
        ));
        std::vector<FileCertificate> files;
        for (const auto &c: all)
            files.push_back(std::get<FileCertificate>(c));
        return files;
    }

    /// @brief issues the next identity, inside the renewal window of the current one.
    void renew() {
        const auto current = ASSERT_NIL_P(this->store->current());
        this->now = current.certificate.not_after() - 30 * x::telem::DAY;
        ASSERT_NIL(this->store->renew());
    }
};
}

/// @brief it should save no record for a Store with no identity.
TEST_F(RecordsTest, SyncEmpty) {
    auto records = this->records();
    ASSERT_NIL(records.sync("Plant"));
    ASSERT_TRUE(this->on_rack().empty());
}

/// @brief it should save the current identity as a file certificate on the rack, with
/// its details and no private key.
TEST_F(RecordsTest, SyncCurrent) {
    const auto id = ASSERT_NIL_P(this->store->current());
    auto records = this->records();
    ASSERT_NIL(records.sync("Plant"));
    const auto files = this->on_rack();
    ASSERT_EQ(files.size(), 1);
    ASSERT_EQ(files[0].name, "Plant OPC UA Driver certificate");
    ASSERT_EQ(files[0].certificate_path, this->store->cert_path(id).string());
    ASSERT_FALSE(files[0].private_key_file.has_value());
    ASSERT_TRUE(files[0].details.has_value());
    ASSERT_EQ(files[0].details.value().sha256, id.certificate.sha256());
    ASSERT_EQ(files[0].details.value().not_after, id.certificate.not_after());
}

/// @brief it should save the next identity beside the current one, then keep only the
/// switched one under the current name after promote.
TEST_F(RecordsTest, SyncRenewal) {
    this->renew();
    auto records = this->records();
    ASSERT_NIL(records.sync("Plant"));
    const auto ids = ASSERT_NIL_P(this->store->list());
    auto files = this->on_rack();
    ASSERT_EQ(files.size(), 2);
    const auto next = std::ranges::find(
        files,
        this->store->cert_path(ids[1]).string(),
        &FileCertificate::certificate_path
    );
    ASSERT_NE(next, files.end());
    ASSERT_EQ(next->name, "Plant next OPC UA Driver certificate");
    const auto next_key = next->key;
    ASSERT_NIL(this->store->promote());
    ASSERT_NIL(records.sync("Plant"));
    files = this->on_rack();
    ASSERT_EQ(files.size(), 1);
    ASSERT_EQ(files[0].key, next_key);
    ASSERT_EQ(files[0].name, "Plant OPC UA Driver certificate");
}

/// @brief it should rename the record when the rack is renamed.
TEST_F(RecordsTest, SyncRackRename) {
    ASSERT_NIL_P(this->store->current());
    auto records = this->records();
    ASSERT_NIL(records.sync("Plant"));
    ASSERT_NIL(records.sync("Line 2"));
    const auto files = this->on_rack();
    ASSERT_EQ(files.size(), 1);
    ASSERT_EQ(files[0].name, "Line 2 OPC UA Driver certificate");
}

/// @brief it should replace the records after a regenerate.
TEST_F(RecordsTest, SyncRegenerate) {
    this->renew();
    auto records = this->records();
    ASSERT_NIL(records.sync("Plant"));
    const auto id = ASSERT_NIL_P(this->store->regenerate());
    ASSERT_NIL(records.sync("Plant"));
    const auto files = this->on_rack();
    ASSERT_EQ(files.size(), 1);
    ASSERT_EQ(files[0].certificate_path, this->store->cert_path(id).string());
}

/// @brief it should call the Core again only after a restart when nothing changed, so a
/// deleted record comes back at the next start.
TEST_F(RecordsTest, SyncRestoresAtStart) {
    ASSERT_NIL_P(this->store->current());
    auto records = this->records();
    ASSERT_NIL(records.sync("Plant"));
    ASSERT_NIL(this->client.certificates.del({this->on_rack()[0].key}));
    ASSERT_NIL(records.sync("Plant"));
    ASSERT_TRUE(this->on_rack().empty());
    auto restarted = this->records();
    ASSERT_NIL(restarted.sync("Plant"));
    ASSERT_EQ(this->on_rack().size(), 1);
}

/// @brief it should keep one record of a file that has two, as after a save whose reply
/// was lost.
TEST_F(RecordsTest, SyncDeletesDuplicate) {
    ASSERT_NIL_P(this->store->current());
    auto records = this->records();
    ASSERT_NIL(records.sync("Plant"));
    synnax::certificate::Certificate duplicate = this->on_rack().at(0);
    std::get<FileCertificate>(duplicate).key = x::uuid::UUID();
    ASSERT_NIL(this->client.certificates.set(duplicate));
    ASSERT_EQ(this->on_rack().size(), 2);
    auto restarted = this->records();
    ASSERT_NIL(restarted.sync("Plant"));
    ASSERT_EQ(this->on_rack().size(), 1);
}

/// @brief it should delete the records of a Store in an earlier state directory.
TEST_F(RecordsTest, SyncDeletesMovedRecords) {
    ASSERT_NIL_P(this->store->current());
    auto records = this->records();
    ASSERT_NIL(records.sync("Plant"));
    const auto moved = std::make_shared<Store>(StoreConfig{.dir = this->dir / "moved"});
    const auto id = ASSERT_NIL_P(moved->current());
    Records restarted(moved, this->as_rack, this->rack.key);
    ASSERT_NIL(restarted.sync("Plant"));
    const auto files = this->on_rack();
    ASSERT_EQ(files.size(), 1);
    ASSERT_EQ(files[0].certificate_path, moved->cert_path(id).string());
}

/// @brief it should leave a file certificate on the rack outside the Store's folder.
TEST_F(RecordsTest, SyncKeepsOtherRecords) {
    synnax::certificate::Certificate other = FileCertificate{};
    auto &file = std::get<FileCertificate>(other);
    file.name = "Plant CA";
    file.certificate_path = "/etc/pki/ca.pem";
    file.rack = this->rack.key;
    ASSERT_NIL(this->client.certificates.set(other));
    ASSERT_NIL_P(this->store->current());
    auto records = this->records();
    ASSERT_NIL(records.sync("Plant"));
    ASSERT_EQ(this->on_rack().size(), 2);
    ASSERT_NIL_P(this->store->regenerate());
    ASSERT_NIL(records.sync("Plant"));
    const auto files = this->on_rack();
    ASSERT_EQ(files.size(), 2);
    ASSERT_NE(
        std::ranges::find(files, "Plant CA", &FileCertificate::name),
        files.end()
    );
}
}
