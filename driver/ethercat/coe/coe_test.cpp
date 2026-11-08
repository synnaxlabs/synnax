// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <map>

#include "gtest/gtest.h"

#include "x/cpp/testutil/testutil.h"

#include "driver/ethercat/coe/coe.h"
#include "driver/ethercat/errors/errors.h"

namespace driver::ethercat::coe {
namespace {
using Objects = std::map<std::pair<uint16_t, uint8_t>, uint32_t>;

/// @brief returns a read of objects that aborts with code for each object in aborts,
/// and as a missing object for any other object it does not hold.
Read reader(const Objects &objects, const Objects &aborts = {}) {
    return [objects, aborts](const uint16_t index, const uint8_t sub_index) {
        if (const auto it = objects.find({index, sub_index}); it != objects.end())
            return std::pair{it->second, x::errors::NIL};
        const auto abort = aborts.find({index, sub_index});
        return std::pair{
            uint32_t{0},
            read_error(
                index,
                sub_index,
                abort == aborts.end() ? 0x06020000 : abort->second
            )
        };
    };
}

/// @brief a slave whose SM2 carries RxPDO 0x1600 with one 16-bit object, and whose
/// SM3 carries TxPDO 0x1A00 with a 1-bit object, 7 bits of padding, and a 16-bit
/// object.
const Objects LAYOUT{
    {{0x1C00, 0}, 4},
    {{0x1C00, 3}, 3},
    {{0x1C00, 4}, 4},
    {{0x1C12, 0}, 1},
    {{0x1C12, 1}, 0x1600},
    {{0x1600, 0}, 1},
    {{0x1600, 1}, 0x70000110},
    {{0x1C13, 0}, 1},
    {{0x1C13, 1}, 0x1A00},
    {{0x1A00, 0}, 3},
    {{0x1A00, 1}, 0x60000101},
    {{0x1A00, 2}, 0x00000007},
    {{0x1A00, 3}, 0x60001110},
};

Objects with(Objects objects, const Objects &changes) {
    for (const auto &[key, value]: changes)
        objects[key] = value;
    return objects;
}

Objects without(Objects objects, const std::pair<uint16_t, uint8_t> &key) {
    objects.erase(key);
    return objects;
}
}

/// @brief it should read the PDOs that each process data sync manager carries and the
/// objects each PDO maps.
TEST(ReadLayoutTest, ReadsThePDOsOfEachSyncManager) {
    const auto syncs = ASSERT_NIL_P(read_layout(reader(LAYOUT), 8, OnError::FAIL));
    ASSERT_EQ(syncs.size(), 2);
    EXPECT_EQ(syncs[0].index, 2);
    EXPECT_FALSE(syncs[0].is_input);
    ASSERT_EQ(syncs[0].pdos.size(), 1);
    EXPECT_EQ(syncs[0].pdos[0].index, 0x1600);
    ASSERT_EQ(syncs[0].pdos[0].entries.size(), 1);
    EXPECT_EQ(syncs[0].pdos[0].entries[0].index, 0x7000);
    EXPECT_EQ(syncs[0].pdos[0].entries[0].sub_index, 1);
    EXPECT_EQ(syncs[0].pdos[0].entries[0].bit_length, 16);
    EXPECT_EQ(syncs[1].index, 3);
    EXPECT_TRUE(syncs[1].is_input);
    ASSERT_EQ(syncs[1].pdos.size(), 1);
    const auto &entries = syncs[1].pdos[0].entries;
    ASSERT_EQ(entries.size(), 3);
    EXPECT_EQ(entries[1].index, 0);
    EXPECT_EQ(entries[1].bit_length, 7);
    EXPECT_EQ(entries[2].sub_index, 17);
}

/// @brief it should return no sync managers for a slave without the sync manager type
/// object, whose layout its EEPROM lists.
TEST(ReadLayoutTest, ReturnsNoSyncManagersWithoutTheTypeObject) {
    const auto syncs = ASSERT_NIL_P(
        read_layout(reader(without(LAYOUT, {0x1C00, 0})), 8, OnError::FAIL)
    );
    EXPECT_TRUE(syncs.empty());
}

/// @brief it should mark a sync manager without a PDO assign object as fixed.
TEST(ReadLayoutTest, MarksASyncManagerWithoutAnAssignObjectAsFixed) {
    const auto syncs = ASSERT_NIL_P(
        read_layout(reader(without(LAYOUT, {0x1C12, 0})), 8, OnError::FAIL)
    );
    ASSERT_EQ(syncs.size(), 2);
    EXPECT_TRUE(syncs[0].fixed);
    EXPECT_TRUE(syncs[0].pdos.empty());
    EXPECT_FALSE(syncs[1].fixed);
}

/// @brief it should fail with the abort reason when the slave refuses a read.
TEST(ReadLayoutTest, FailsWithTheAbortReasonOfARefusedRead) {
    const auto res = read_layout(
        reader(without(LAYOUT, {0x1A00, 3}), {{{0x1A00, 3}, 0x06010000}}),
        8,
        OnError::FAIL
    );
    ASSERT_OCCURRED_AS_P(res, errors::SDO_READ_ERROR);
    EXPECT_EQ(
        res.second.data,
        "failed to read 0x1A00:03: Unsupported access to an object"
    );
}

/// @brief it should fail when the slave assigns a PDO that it does not have.
TEST(ReadLayoutTest, FailsOnAnAssignedPDOThatIsMissing) {
    const auto res = read_layout(
        reader(without(LAYOUT, {0x1A00, 0})),
        8,
        OnError::FAIL
    );
    ASSERT_OCCURRED_AS_P(res, OBJECT_MISSING);
    EXPECT_EQ(
        res.second.data,
        "failed to read 0x1A00:00: Object does not exist in the object dictionary"
    );
}

/// @brief it should skip a sync manager whose type it cannot read and keep the others,
/// as SOEM does.
TEST(ReadLayoutTest, SkipsASyncManagerWhoseTypeReadFails) {
    const auto res = read_layout(
        reader(without(LAYOUT, {0x1C00, 4}), {{{0x1C00, 4}, 0x06010000}}),
        8,
        OnError::SKIP
    );
    ASSERT_OCCURRED_AS_P(res, errors::SDO_READ_ERROR);
    EXPECT_EQ(
        res.second.data,
        "failed to read 0x1C00:04: Unsupported access to an object"
    );
    ASSERT_EQ(res.first.size(), 1);
    EXPECT_EQ(res.first[0].index, 2);
}

/// @brief it should skip the objects of a PDO it cannot read and keep the other PDOs,
/// as SOEM does.
TEST(ReadLayoutTest, SkipsAPDOItCannotRead) {
    const auto res = read_layout(
        reader(without(LAYOUT, {0x1A00, 0})),
        8,
        OnError::SKIP
    );
    ASSERT_OCCURRED_AS_P(res, OBJECT_MISSING);
    ASSERT_EQ(res.first.size(), 2);
    EXPECT_EQ(res.first[0].pdos[0].entries.size(), 1);
    EXPECT_TRUE(res.first[1].pdos[0].entries.empty());
}

/// @brief it should skip an object mapping it cannot read and keep the others, as SOEM
/// does.
TEST(ReadLayoutTest, SkipsAnObjectMappingItCannotRead) {
    const auto res = read_layout(
        reader(without(LAYOUT, {0x1A00, 1})),
        8,
        OnError::SKIP
    );
    ASSERT_OCCURRED_AS_P(res, OBJECT_MISSING);
    ASSERT_EQ(res.first.size(), 2);
    const auto &entries = res.first[1].pdos[0].entries;
    ASSERT_EQ(entries.size(), 2);
    EXPECT_EQ(entries[0].bit_length, 7);
    EXPECT_EQ(entries[1].sub_index, 17);
}

/// @brief it should read a count above the maximum as the maximum, as SOEM does.
TEST(ReadLayoutTest, ReadsACountAboveTheMaximumAsTheMaximum) {
    const auto res = read_layout(
        reader(with(LAYOUT, {{{0x1C13, 0}, 300}})),
        8,
        OnError::SKIP
    );
    ASSERT_OCCURRED_AS_P(res, errors::PDO_DISCOVERY_ERROR);
    EXPECT_EQ(
        res.second.data,
        "0x1C13:00 counts 300 entries, above the maximum of 254"
    );
    ASSERT_EQ(res.first.size(), 2);
    ASSERT_EQ(res.first[1].pdos.size(), 1);
    EXPECT_EQ(res.first[1].pdos[0].index, 0x1A00);
}

/// @brief it should fail when a PDO counts more objects than it can map.
TEST(ReadLayoutTest, FailsOnAPDOThatCountsTooManyObjects) {
    const auto res = read_layout(
        reader(with(LAYOUT, {{{0x1A00, 0}, 0xFFFFFFFF}})),
        8,
        OnError::FAIL
    );
    ASSERT_OCCURRED_AS_P(res, errors::PDO_DISCOVERY_ERROR);
    EXPECT_EQ(
        res.second.data,
        "0x1A00:00 counts 4294967295 entries, above the maximum of 254"
    );
}

/// @brief it should read only the sync managers the master configures when the slave
/// counts more.
TEST(ReadLayoutTest, ReadsOnlyTheSyncManagersTheMasterConfigures) {
    const auto syncs = ASSERT_NIL_P(
        read_layout(reader(with(LAYOUT, {{{0x1C00, 0}, 9}})), 4, OnError::FAIL)
    );
    ASSERT_EQ(syncs.size(), 2);
    EXPECT_EQ(syncs[1].index, 3);
}

/// @brief it should read SM2 as outputs and SM3 as inputs when the slave gives them
/// type 0, as SOEM maps them.
TEST(ReadLayoutTest, ReadsSyncManagersOfTypeZeroAsSOEMMapsThem) {
    const auto syncs = ASSERT_NIL_P(read_layout(
        reader(with(LAYOUT, {{{0x1C00, 3}, 0}, {{0x1C00, 4}, 0}})),
        8,
        OnError::FAIL
    ));
    ASSERT_EQ(syncs.size(), 2);
    EXPECT_FALSE(syncs[0].is_input);
    EXPECT_TRUE(syncs[1].is_input);
}

/// @brief it should read types one too low when SM2 gives the mailbox type, as SOEM
/// maps them.
TEST(ReadLayoutTest, ShiftsTypesThatStartAtTheMailbox) {
    const auto syncs = ASSERT_NIL_P(read_layout(
        reader(with(LAYOUT, {{{0x1C00, 3}, 2}, {{0x1C00, 4}, 3}})),
        8,
        OnError::FAIL
    ));
    ASSERT_EQ(syncs.size(), 2);
    EXPECT_FALSE(syncs[0].is_input);
    EXPECT_TRUE(syncs[1].is_input);
}

/// @brief it should skip an empty sub-index of an assign object.
TEST(ReadLayoutTest, SkipsAnEmptyAssignment) {
    const auto syncs = ASSERT_NIL_P(read_layout(
        reader(with(LAYOUT, {{{0x1C13, 0}, 2}, {{0x1C13, 2}, 0}})),
        8,
        OnError::FAIL
    ));
    ASSERT_EQ(syncs[1].pdos.size(), 1);
}
}
