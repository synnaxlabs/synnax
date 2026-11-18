// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

extern "C" {
#include "osal.h"
#include "oshw.h"
}

#include <array>
#include <cstdio>
#include <cstring>
#include <set>
#include <string>
#include <vector>

#include "gtest/gtest.h"
#include <arpa/inet.h>
#include <pcap.h>

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

/// @brief the source MAC of a frame that passed the slaves, which change the MAC.
constexpr std::array<uint16, 3> SLAVE_MAC = {0x0202, 0x0000, 0x0001};

/// @brief a port that reads the packets of an in-memory capture.
class OshwReceiveTest : public OshwBufferTest {
protected:
    std::vector<uint8_t> capture;

    void add(const std::vector<uint8_t> &bytes) {
        this->capture.insert(this->capture.end(), bytes.begin(), bytes.end());
    }

    void add32(const uint32_t value) {
        this->add({
            static_cast<uint8_t>(value),
            static_cast<uint8_t>(value >> 8),
            static_cast<uint8_t>(value >> 16),
            static_cast<uint8_t>(value >> 24),
        });
    }

    void SetUp() override {
        OshwBufferTest::SetUp();
        auto &stack = this->port.stack;
        stack.pcap_handle = &this->port.pcap_handle;
        stack.txbuf = &this->port.txbuf;
        stack.txbuflength = &this->port.txbuflength;
        stack.tempbuf = &this->port.tempinbuf;
        stack.rxbuf = &this->port.rxbuf;
        stack.rxbufstat = &this->port.rxbufstat;
        stack.rxsa = &this->port.rxsa;
        // The capture header: version 2.4, a 65535 byte snap length, and Ethernet.
        this->add32(0xa1b2c3d4);
        this->add32(0x00040002);
        this->add32(0);
        this->add32(0);
        this->add32(65535);
        this->add32(1);
    }

    void TearDown() override {
        if (this->port.pcap_handle != nullptr) pcap_close(this->port.pcap_handle);
        OshwBufferTest::TearDown();
    }

    /// @brief adds a frame of one datagram with 2 data bytes for the index, from the
    /// source MAC with the words mac.
    void add_frame(const uint8 idx, const uint16 wkc, const std::array<uint16, 3> mac) {
        std::vector<uint8_t> frame(ETH_HEADERSIZE + EC_HEADERSIZE + 2 + EC_WKCSIZE);
        std::memset(frame.data(), 0xff, 6);
        for (int i = 0; i < 3; i++) {
            frame[6 + 2 * i] = static_cast<uint8_t>(mac[i] >> 8);
            frame[7 + 2 * i] = static_cast<uint8_t>(mac[i]);
        }
        frame[12] = ETH_P_ECAT >> 8;
        frame[13] = ETH_P_ECAT & 0xff;
        const uint16 length = EC_ECATTYPE + EC_HEADERSIZE + 2;
        frame[ETH_HEADERSIZE] = static_cast<uint8_t>(length);
        frame[ETH_HEADERSIZE + 1] = static_cast<uint8_t>(length >> 8);
        frame[ETH_HEADERSIZE + 3] = idx;
        frame[frame.size() - 2] = static_cast<uint8_t>(wkc);
        frame[frame.size() - 1] = static_cast<uint8_t>(wkc >> 8);
        this->add32(0);
        this->add32(0);
        this->add32(static_cast<uint32_t>(frame.size()));
        this->add32(static_cast<uint32_t>(frame.size()));
        this->add(frame);
        this->port.txbuflength[idx] = static_cast<int>(frame.size());
    }

    void open() {
        FILE *file = fmemopen(this->capture.data(), this->capture.size(), "rb");
        ASSERT_NE(file, nullptr);
        char err[PCAP_ERRBUF_SIZE];
        this->port.pcap_handle = pcap_fopen_offline(file, err);
        ASSERT_NE(this->port.pcap_handle, nullptr) << err;
    }
};

/// @brief it should read past the echo of a frame and a frame of another index to the
/// frame that it waits for, even with no time to wait.
TEST_F(OshwReceiveTest, WaitReadsPastOtherPacketsWithNoTimeout) {
    constexpr uint8 idx = 3;
    this->add_frame(idx, 0, {priMAC[0], priMAC[1], priMAC[2]});
    this->add_frame(5, 1, SLAVE_MAC);
    this->add_frame(idx, 7, SLAVE_MAC);
    this->port.rxbufstat[idx] = EC_BUF_TX;
    ASSERT_NO_FATAL_FAILURE(this->open());
    EXPECT_EQ(ecx_waitinframe(&this->port, idx, 0), 7);
}

/// @brief it should return no frame when the capture holds none for the index.
TEST_F(OshwReceiveTest, WaitReturnsNoFrameWhenNoneCameIn) {
    constexpr uint8 idx = 3;
    this->add_frame(5, 1, SLAVE_MAC);
    this->port.rxbufstat[idx] = EC_BUF_TX;
    ASSERT_NO_FATAL_FAILURE(this->open());
    EXPECT_EQ(ecx_waitinframe(&this->port, idx, 0), EC_NOFRAME);
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

/// @brief it should leave out every adapter that libpcap reports as wireless.
TEST_F(OshwAdapterTest, FindAdaptersExcludesWirelessAdapters) {
    std::set<std::string> wireless;
    pcap_if_t *devs = nullptr;
    char errbuf[PCAP_ERRBUF_SIZE];
    ASSERT_EQ(pcap_findalldevs(&devs, errbuf), 0) << errbuf;
    for (pcap_if_t *d = devs; d != nullptr; d = d->next)
        if (d->flags & PCAP_IF_WIRELESS) wireless.insert(d->name);
    pcap_freealldevs(devs);
    if (wireless.empty()) GTEST_SKIP() << "this host has no wireless adapter";
    ec_adaptert *adapters = oshw_find_adapters();
    for (ec_adaptert *a = adapters; a != nullptr; a = a->next)
        EXPECT_FALSE(wireless.contains(a->name)) << a->name;
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
