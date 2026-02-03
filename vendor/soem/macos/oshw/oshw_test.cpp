// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.

extern "C" {
#include "osal.h"
#include "oshw.h"
}

#include <cstring>
#include <string>

#include <arpa/inet.h>
#include <gtest/gtest.h>

/// @brief Test fixture for byte order conversion tests.
class OshwByteOrderTest : public ::testing::Test {};

/// @brief Test fixture for ethernet header setup tests.
class OshwHeaderTest : public ::testing::Test {
protected:
    ec_bufT buffer;

    void SetUp() override { std::memset(this->buffer, 0, sizeof(this->buffer)); }
};

/// @brief Test fixture for buffer management tests.
class OshwBufferTest : public ::testing::Test {
protected:
    ecx_portt port;

    void SetUp() override {
        std::memset(&this->port, 0, sizeof(this->port));
        pthread_mutexattr_t mutexattr;
        pthread_mutexattr_init(&mutexattr);
        pthread_mutex_init(&this->port.getindex_mutex, &mutexattr);
        pthread_mutex_init(&this->port.tx_mutex, &mutexattr);
        pthread_mutex_init(&this->port.rx_mutex, &mutexattr);
        for (int i = 0; i < EC_MAXBUF; i++)
            this->port.rxbufstat[i] = EC_BUF_EMPTY;
    }

    void TearDown() override {
        pthread_mutex_destroy(&this->port.getindex_mutex);
        pthread_mutex_destroy(&this->port.tx_mutex);
        pthread_mutex_destroy(&this->port.rx_mutex);
    }
};

/// @brief Test fixture for adapter discovery tests.
class OshwAdapterTest : public ::testing::Test {};

/// @brief Test fixture for NIC setup and teardown tests.
class OshwNicTest : public ::testing::Test {
protected:
    ecx_portt port;
    std::string interfaceName;

    void SetUp() override {
        std::memset(&this->port, 0, sizeof(this->port));
        ec_adaptert *adapters = oshw_find_adapters();
        if (adapters != nullptr) {
            this->interfaceName = adapters->name;
            oshw_free_adapters(adapters);
        }
    }

    void TearDown() override { ecx_closenic(&this->port); }
};

TEST_F(OshwByteOrderTest, HtonsConvertsKnownValue) {
    uint16 host = 0x1234;
    uint16 network = oshw_htons(host);
    EXPECT_EQ(network, htons(host));
}

TEST_F(OshwByteOrderTest, NtohsConvertsKnownValue) {
    uint16 network = 0x1234;
    uint16 host = oshw_ntohs(network);
    EXPECT_EQ(host, ntohs(network));
}

TEST_F(OshwByteOrderTest, HtonsZeroIsZero) {
    EXPECT_EQ(oshw_htons(0), static_cast<uint16>(0));
}

TEST_F(OshwByteOrderTest, NtohsZeroIsZero) {
    EXPECT_EQ(oshw_ntohs(0), static_cast<uint16>(0));
}

TEST_F(OshwByteOrderTest, HtonsNtohsRoundTrip) {
    uint16 original = 0xABCD;
    uint16 result = oshw_ntohs(oshw_htons(original));
    EXPECT_EQ(result, original);
}

TEST_F(OshwByteOrderTest, NtohsHtonsRoundTrip) {
    uint16 original = 0x5678;
    uint16 result = oshw_htons(oshw_ntohs(original));
    EXPECT_EQ(result, original);
}

TEST_F(OshwHeaderTest, SetupHeaderSetsDestinationBroadcast) {
    ec_setupheader(this->buffer);
    ec_etherheadert *header = reinterpret_cast<ec_etherheadert *>(this->buffer);
    EXPECT_EQ(header->da0, htons(0xffff));
    EXPECT_EQ(header->da1, htons(0xffff));
    EXPECT_EQ(header->da2, htons(0xffff));
}

TEST_F(OshwHeaderTest, SetupHeaderSetsSourceMac) {
    ec_setupheader(this->buffer);
    ec_etherheadert *header = reinterpret_cast<ec_etherheadert *>(this->buffer);
    EXPECT_EQ(header->sa0, htons(priMAC[0]));
    EXPECT_EQ(header->sa1, htons(priMAC[1]));
    EXPECT_EQ(header->sa2, htons(priMAC[2]));
}

TEST_F(OshwHeaderTest, SetupHeaderSetsEthertypeEcat) {
    ec_setupheader(this->buffer);
    ec_etherheadert *header = reinterpret_cast<ec_etherheadert *>(this->buffer);
    EXPECT_EQ(header->etype, htons(ETH_P_ECAT));
}

TEST_F(OshwHeaderTest, SetupHeaderProducesCorrectSize) {
    ec_setupheader(this->buffer);
    EXPECT_EQ(sizeof(ec_etherheadert), static_cast<size_t>(14));
}

TEST_F(OshwBufferTest, GetIndexReturnsZeroOnFirstCall) {
    this->port.lastidx = EC_MAXBUF - 1;
    uint8 idx = ecx_getindex(&this->port);
    EXPECT_EQ(idx, 0);
}

TEST_F(OshwBufferTest, GetIndexIncrementsLastIdx) {
    this->port.lastidx = 0;
    uint8 idx = ecx_getindex(&this->port);
    EXPECT_EQ(this->port.lastidx, idx);
}

TEST_F(OshwBufferTest, GetIndexWrapsAroundAtMaxBuf) {
    this->port.lastidx = EC_MAXBUF - 2;
    ecx_getindex(&this->port);
    uint8 idx = ecx_getindex(&this->port);
    EXPECT_EQ(idx, 0);
}

TEST_F(OshwBufferTest, GetIndexSkipsNonEmptyBuffers) {
    for (int i = 0; i < EC_MAXBUF - 1; i++)
        this->port.rxbufstat[i] = EC_BUF_TX;
    this->port.rxbufstat[EC_MAXBUF - 1] = EC_BUF_EMPTY;
    this->port.lastidx = 0;
    uint8 idx = ecx_getindex(&this->port);
    EXPECT_EQ(idx, EC_MAXBUF - 1);
}

TEST_F(OshwBufferTest, GetIndexSetsBufferToAlloc) {
    this->port.lastidx = 0;
    uint8 idx = ecx_getindex(&this->port);
    EXPECT_EQ(this->port.rxbufstat[idx], EC_BUF_ALLOC);
}

TEST_F(OshwBufferTest, GetIndexReturnsSequentialIndices) {
    uint8 first = ecx_getindex(&this->port);
    this->port.rxbufstat[first] = EC_BUF_EMPTY;
    uint8 second = ecx_getindex(&this->port);
    EXPECT_EQ(second, (first + 1) % EC_MAXBUF);
}

TEST_F(OshwBufferTest, SetBufStatChangesStatus) {
    uint8 idx = 0;
    ecx_setbufstat(&this->port, idx, EC_BUF_TX);
    EXPECT_EQ(this->port.rxbufstat[idx], EC_BUF_TX);
}

TEST_F(OshwBufferTest, SetBufStatSetsComplete) {
    uint8 idx = 5;
    ecx_setbufstat(&this->port, idx, EC_BUF_COMPLETE);
    EXPECT_EQ(this->port.rxbufstat[idx], EC_BUF_COMPLETE);
}

TEST_F(OshwBufferTest, SetBufStatSetsEmpty) {
    uint8 idx = 3;
    this->port.rxbufstat[idx] = EC_BUF_COMPLETE;
    ecx_setbufstat(&this->port, idx, EC_BUF_EMPTY);
    EXPECT_EQ(this->port.rxbufstat[idx], EC_BUF_EMPTY);
}

TEST_F(OshwAdapterTest, FindAdaptersReturnsNonNull) {
    ec_adaptert *adapters = oshw_find_adapters();
    EXPECT_NE(adapters, nullptr);
    oshw_free_adapters(adapters);
}

TEST_F(OshwAdapterTest, FindAdaptersHasNonEmptyName) {
    ec_adaptert *adapters = oshw_find_adapters();
    ASSERT_NE(adapters, nullptr);
    EXPECT_GT(strlen(adapters->name), static_cast<size_t>(0));
    oshw_free_adapters(adapters);
}

TEST_F(OshwAdapterTest, FindAdaptersHasNonEmptyDesc) {
    ec_adaptert *adapters = oshw_find_adapters();
    ASSERT_NE(adapters, nullptr);
    EXPECT_GT(strlen(adapters->desc), static_cast<size_t>(0));
    oshw_free_adapters(adapters);
}

TEST_F(OshwAdapterTest, FindAdaptersContainsLoopback) {
    ec_adaptert *adapters = oshw_find_adapters();
    ASSERT_NE(adapters, nullptr);
    bool found_loopback = false;
    for (ec_adaptert *a = adapters; a != nullptr; a = a->next) {
        if (strstr(a->name, "lo") != nullptr) {
            found_loopback = true;
            break;
        }
    }
    EXPECT_TRUE(found_loopback);
    oshw_free_adapters(adapters);
}

TEST_F(OshwAdapterTest, FreeAdaptersAcceptsNull) {
    oshw_free_adapters(nullptr);
}

TEST_F(OshwAdapterTest, AdapterNameWithinMaxLength) {
    ec_adaptert *adapters = oshw_find_adapters();
    ASSERT_NE(adapters, nullptr);
    for (ec_adaptert *a = adapters; a != nullptr; a = a->next) {
        EXPECT_LT(strlen(a->name), static_cast<size_t>(EC_MAXLEN_ADAPTERNAME));
    }
    oshw_free_adapters(adapters);
}

TEST_F(OshwAdapterTest, AdapterDescWithinMaxLength) {
    ec_adaptert *adapters = oshw_find_adapters();
    ASSERT_NE(adapters, nullptr);
    for (ec_adaptert *a = adapters; a != nullptr; a = a->next) {
        EXPECT_LT(strlen(a->desc), static_cast<size_t>(EC_MAXLEN_ADAPTERNAME));
    }
    oshw_free_adapters(adapters);
}

TEST_F(OshwAdapterTest, AdapterListIsProperlyLinked) {
    ec_adaptert *adapters = oshw_find_adapters();
    ASSERT_NE(adapters, nullptr);
    int count = 0;
    for (ec_adaptert *a = adapters; a != nullptr; a = a->next) {
        count++;
        if (count > 100) { FAIL() << "Possible infinite loop in adapter list"; }
    }
    EXPECT_GT(count, 0);
    oshw_free_adapters(adapters);
}

TEST_F(OshwNicTest, SetupNicFailsWithInvalidInterface) {
    int result = ecx_setupnic(&this->port, "invalid_interface_xyz", 0);
    EXPECT_EQ(result, 0);
}

TEST_F(OshwNicTest, SetupNicFailsWithEmptyName) {
    int result = ecx_setupnic(&this->port, "", 0);
    EXPECT_EQ(result, 0);
}

TEST_F(OshwNicTest, SetupNicSecondaryFailsWithoutRedport) {
    this->port.redport = nullptr;
    int result = ecx_setupnic(&this->port, "lo0", 1);
    EXPECT_EQ(result, 0);
}

TEST_F(OshwNicTest, CloseNicHandlesNullHandle) {
    this->port.pcap_handle = nullptr;
    this->port.redport = nullptr;
    int result = ecx_closenic(&this->port);
    EXPECT_EQ(result, 0);
}

TEST_F(OshwNicTest, CloseNicReturnsZero) {
    int result = ecx_closenic(&this->port);
    EXPECT_EQ(result, 0);
}

TEST_F(OshwNicTest, SetupNicInitializesBufferStatus) {
    if (this->interfaceName.empty()) GTEST_SKIP() << "No network interface available";
    int result = ecx_setupnic(&this->port, this->interfaceName.c_str(), 0);
    if (result == 0)
        GTEST_SKIP() << "Could not open interface (may require elevated privileges)";
    for (int i = 0; i < EC_MAXBUF; i++) {
        EXPECT_EQ(this->port.rxbufstat[i], EC_BUF_EMPTY);
    }
}

TEST_F(OshwNicTest, SetupNicInitializesLastIdx) {
    if (this->interfaceName.empty()) GTEST_SKIP() << "No network interface available";
    int result = ecx_setupnic(&this->port, this->interfaceName.c_str(), 0);
    if (result == 0)
        GTEST_SKIP() << "Could not open interface (may require elevated privileges)";
    EXPECT_EQ(this->port.lastidx, 0);
}

TEST_F(OshwNicTest, SetupNicSetsStackPointers) {
    if (this->interfaceName.empty()) GTEST_SKIP() << "No network interface available";
    int result = ecx_setupnic(&this->port, this->interfaceName.c_str(), 0);
    if (result == 0)
        GTEST_SKIP() << "Could not open interface (may require elevated privileges)";
    EXPECT_EQ(this->port.stack.pcap_handle, &this->port.pcap_handle);
    EXPECT_EQ(this->port.stack.txbuf, &this->port.txbuf);
    EXPECT_EQ(this->port.stack.rxbuf, &this->port.rxbuf);
}

TEST_F(OshwNicTest, SetupNicSetsPcapHandle) {
    if (this->interfaceName.empty()) GTEST_SKIP() << "No network interface available";
    int result = ecx_setupnic(&this->port, this->interfaceName.c_str(), 0);
    if (result == 0)
        GTEST_SKIP() << "Could not open interface (may require elevated privileges)";
    EXPECT_NE(this->port.pcap_handle, nullptr);
}
