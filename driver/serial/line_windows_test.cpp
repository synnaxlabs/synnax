// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/serial/line.h"

namespace driver::serial::line {
TEST(Line, AppliesSevenDataBitsOddParityTwoStopBitsAndHardwareFlowControl) {
    synnax::serial::Properties p;
    p.baud_rate = 19200;
    p.data_bits = 7;
    p.parity = synnax::serial::PARITY_ODD_;
    p.stop_bits = synnax::serial::STOP_BITS_TWO;
    p.flow_control = synnax::serial::FLOW_CONTROL_HARDWARE;
    Settings s{};
    ASSERT_NIL(apply(p, s));
    EXPECT_EQ(s.BaudRate, 19200u);
    EXPECT_EQ(s.ByteSize, 7);
    EXPECT_EQ(s.fParity, static_cast<DWORD>(TRUE));
    EXPECT_EQ(s.Parity, ODDPARITY);
    EXPECT_EQ(s.StopBits, TWOSTOPBITS);
    EXPECT_EQ(s.fOutxCtsFlow, static_cast<DWORD>(TRUE));
    EXPECT_EQ(s.fRtsControl, static_cast<DWORD>(RTS_CONTROL_HANDSHAKE));
    EXPECT_EQ(s.fOutX, static_cast<DWORD>(FALSE));
    EXPECT_EQ(s.fInX, static_cast<DWORD>(FALSE));
}

TEST(Line, AppliesTheDefaults) {
    Settings s{};
    s.fParity = TRUE;
    s.Parity = MARKPARITY;
    s.StopBits = TWOSTOPBITS;
    s.fOutxCtsFlow = TRUE;
    s.fOutxDsrFlow = TRUE;
    ASSERT_NIL(apply(synnax::serial::Properties{}, s));
    EXPECT_EQ(s.BaudRate, 9600u);
    EXPECT_EQ(s.ByteSize, 8);
    EXPECT_EQ(s.fParity, static_cast<DWORD>(FALSE));
    EXPECT_EQ(s.Parity, NOPARITY);
    EXPECT_EQ(s.StopBits, ONESTOPBIT);
    EXPECT_EQ(s.fOutxCtsFlow, static_cast<DWORD>(FALSE));
    EXPECT_EQ(s.fOutxDsrFlow, static_cast<DWORD>(FALSE));
    EXPECT_EQ(s.fDtrControl, static_cast<DWORD>(DTR_CONTROL_ENABLE));
    EXPECT_EQ(s.fRtsControl, static_cast<DWORD>(RTS_CONTROL_ENABLE));
}

TEST(Line, AppliesSoftwareFlowControl) {
    synnax::serial::Properties p;
    p.flow_control = synnax::serial::FLOW_CONTROL_SOFTWARE;
    Settings s{};
    ASSERT_NIL(apply(p, s));
    EXPECT_EQ(s.fOutX, static_cast<DWORD>(TRUE));
    EXPECT_EQ(s.fInX, static_cast<DWORD>(TRUE));
    EXPECT_EQ(s.fTXContinueOnXoff, static_cast<DWORD>(TRUE));
    EXPECT_EQ(s.fOutxCtsFlow, static_cast<DWORD>(FALSE));
}

TEST(Line, AppliesMarkAndSpaceParity) {
    synnax::serial::Properties p;
    Settings s{};
    p.parity = synnax::serial::PARITY_MARK_;
    ASSERT_NIL(apply(p, s));
    EXPECT_EQ(s.fParity, static_cast<DWORD>(TRUE));
    EXPECT_EQ(s.Parity, MARKPARITY);
    p.parity = synnax::serial::PARITY_SPACE_;
    ASSERT_NIL(apply(p, s));
    EXPECT_EQ(s.Parity, SPACEPARITY);
}

TEST(Line, AppliesOneAndAHalfStopBits) {
    synnax::serial::Properties p;
    p.stop_bits = synnax::serial::STOP_BITS_ONE_AND_HALF;
    Settings s{};
    ASSERT_NIL(apply(p, s));
    EXPECT_EQ(s.StopBits, ONE5STOPBITS);
}

TEST(Line, TogglesRTSForRS485) {
    synnax::serial::Properties p;
    p.rs485 = true;
    Settings s{};
    ASSERT_NIL(apply(p, s));
    EXPECT_EQ(s.fRtsControl, static_cast<DWORD>(RTS_CONTROL_TOGGLE));
}

TEST(Line, CarriesABaudRateOutsideTheStandardTable) {
    synnax::serial::Properties p;
    p.baud_rate = 250000;
    Settings s{};
    ASSERT_NIL(apply(p, s));
    EXPECT_EQ(s.BaudRate, 250000u);
}
}
