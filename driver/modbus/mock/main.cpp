// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <iostream>
#include <csignal>
#include "driver/modbus/mock/slave.h"

std::atomic<bool> running(true);

void signal_handler(int signal) {
    std::cout << "\nReceived signal " << signal << ", shutting down..." << std::endl;
    running = false;
}

int main() {
    // Set up signal handling for graceful shutdown
    signal(SIGINT, signal_handler);
    signal(SIGTERM, signal_handler);

    // Configure the slave with various channel types
    modbus::mock::SlaveConfig config;
    
    // Coils (read/write bits)
    config.coils[0] = 1;    // ON
    config.coils[1] = 0;    // OFF
    config.coils[2] = 1;    // ON
    
    // Discrete Inputs (read-only bits)
    config.discrete_inputs[0] = 1;    // HIGH
    config.discrete_inputs[1] = 0;    // LOW
    config.discrete_inputs[2] = 1;    // HIGH
    
    // Holding Registers (read/write 16-bit)
    config.holding_registers[0] = 12345;    // Decimal value
    config.holding_registers[1] = 0xABCD;   // Hex value
    config.holding_registers[2] = 0x1234;   // Hex value
    
    // Input Registers (read-only 16-bit)
    config.input_registers[0] = 54321;    // Decimal value
    config.input_registers[1] = 0xDCBA;   // Hex value
    config.input_registers[2] = 0x4321;   // Hex value
    
    // Configure network settings
    config.host = "127.0.0.1";  // localhost
    config.port = 1502;         // standard Modbus TCP port

    try {
        std::cout << "Starting Modbus slave on " << config.host << ":" << config.port << std::endl;
        
        // Create and start the slave
        modbus::mock::Slave slave(config);
        auto err = slave.start();
        if (err) {
            std::cerr << "Failed to start slave: " << err.message() << std::endl;
            return 1;
        }

        // Print configured values
        std::cout << "\nConfigured values:" << std::endl;
        std::cout << "Coils (read/write bits):" << std::endl;
        for (const auto& [addr, value] : config.coils) {
            std::cout << "  Address " << addr << ": " << (value ? "ON" : "OFF") << std::endl;
        }

        std::cout << "\nDiscrete Inputs (read-only bits):" << std::endl;
        for (const auto& [addr, value] : config.discrete_inputs) {
            std::cout << "  Address " << addr << ": " << (value ? "HIGH" : "LOW") << std::endl;
        }

        std::cout << "\nHolding Registers (read/write 16-bit):" << std::endl;
        for (const auto& [addr, value] : config.holding_registers) {
            std::cout << "  Address " << addr << ": " << value 
                      << " (0x" << std::hex << value << std::dec << ")" << std::endl;
        }

        std::cout << "\nInput Registers (read-only 16-bit):" << std::endl;
        for (const auto& [addr, value] : config.input_registers) {
            std::cout << "  Address " << addr << ": " << value 
                      << " (0x" << std::hex << value << std::dec << ")" << std::endl;
        }

        std::cout << "\nSlave is running. Press Ctrl+C to stop." << std::endl;

        // Keep the program running until signal is received
        while (running) {
            std::this_thread::sleep_for(std::chrono::milliseconds(100));
        }

        std::cout << "Stopping slave..." << std::endl;
        slave.stop();
        
    } catch (const std::exception& e) {
        std::cerr << "Error: " << e.what() << std::endl;
        return 1;
    }

    return 0;
} 