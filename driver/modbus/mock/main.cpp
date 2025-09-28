// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <csignal>
#include <iostream>
#include "driver/modbus/mock/slave.h"

std::atomic running(true);

void signal_handler(const int signal) {
    std::cout << "\nReceived signal " << signal << ", shutting down..." << std::endl;
    running = false;
}

int main() {
    // Set up signal handling for graceful shutdown
    signal(SIGINT, signal_handler);
    signal(SIGTERM, signal_handler);

    modbus::mock::SlaveConfig config;

    config.coils[0] = 1; // ON
    config.coils[1] = 0; // OFF
    config.coils[2] = 1; // ON

    config.discrete_inputs[0] = 1; // HIGH
    config.discrete_inputs[1] = 0; // LOW
    config.discrete_inputs[2] = 1; // HIGH

    config.holding_registers[0] = static_cast<float>(100); // Decimal value
    config.holding_registers[4] = static_cast<float>(200); // Decimal value

    config.input_registers[0] = static_cast<float>(54321); // Decimal value

    config.host = "127.0.0.1"; // localhost
    config.port = 1502; // standard Modbus TCP port

    try {
        std::cout << "Starting Modbus slave on " << config.host << ":" << config.port
                  << std::endl;

        // Create and start the slave
        modbus::mock::Slave slave(config);
        if (const auto err = slave.start()) {
            std::cerr << "Failed to start slave: " << err.message() << std::endl;
            return 1;
        }

        std::cout << "\nConfigured values:" << std::endl;
        std::cout << "Coils (read/write bits):" << std::endl;
        for (const auto &[addr, value]: config.coils)
            std::cout << "  Address " << addr << ": " << (value ? "ON" : "OFF")
                      << std::endl;

        std::cout << "\nDiscrete Inputs (read-only bits):" << std::endl;
        for (const auto &[addr, value]: config.discrete_inputs)
            std::cout << "  Address " << addr << ": " << (value ? "HIGH" : "LOW")
                      << std::endl;

        std::cout << "\nHolding Registers (read/write 16-bit):" << std::endl;
        for (const auto &[addr, value]: config.holding_registers)
            std::cout << "  Address " << addr << ": " << telem::to_string(value);

        std::cout << "\nInput Registers (read-only 16-bit):" << std::endl;
        for (const auto &[addr, value]: config.input_registers)
            std::cout << "  Address " << addr << ": " << telem::to_string(value);

        std::cout << "\nSlave is running. Press Ctrl+C to stop." << std::endl;

        while (running)
            std::this_thread::sleep_for(std::chrono::milliseconds(100));

        std::cout << "Stopping slave..." << std::endl;
        slave.stop();

    } catch (const std::exception &e) {
        std::cerr << "Error: " << e.what() << std::endl;
        return 1;
    }

    return 0;
}
