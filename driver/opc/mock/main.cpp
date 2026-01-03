// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <csignal>
#include <iostream>
#include <thread>

#include "driver/opc/mock/server.h"

std::atomic<bool> running(true);

void signal_handler(const int signal) {
    std::cout << "\nReceived signal " << signal << ", shutting down..." << std::endl;
    running = false;
}

int main(int argc, char *argv[]) {
    // Set up signal handling for graceful shutdown
    signal(SIGINT, signal_handler);
    signal(SIGTERM, signal_handler);

    // Parse command line arguments for port
    std::uint16_t port = 4840; // Default OPC UA port
    if (argc > 1) {
        try {
            port = static_cast<std::uint16_t>(std::stoi(argv[1]));
        } catch (const std::exception &e) {
            std::cerr << "Invalid port number: " << argv[1] << std::endl;
            return 1;
        }
    }

    try {
        std::cout << "Starting OPC UA Mock Server on port " << port << std::endl;

        // Create server configuration with comprehensive test nodes
        auto config = mock::ServerConfig::create_default();
        config.port = port;

        // Create and start the server
        mock::Server server(config);
        server.start();

        std::cout << "\nOPC UA Mock Server is running with the following test nodes:"
                  << std::endl;
        std::cout << "Endpoint: opc.tcp://localhost:" << port << std::endl;
        std::cout << "\nAvailable test nodes:" << std::endl;

        for (const auto &node: config.test_nodes) {
            std::cout << "  ns=" << node.ns << ";s=" << node.node_id << " ("
                      << node.description << ")" << std::endl;
        }

        std::cout << "\nExample node IDs for testing:" << std::endl;
        std::cout << "  Boolean: ns=1;s=TestBoolean" << std::endl;
        std::cout << "  Int32:   ns=1;s=TestInt32" << std::endl;
        std::cout << "  Float32: ns=1;s=TestFloat32" << std::endl;
        std::cout << "  String:  ns=1;s=TestString" << std::endl;
        std::cout << "  GUID:    ns=1;s=TestGuid" << std::endl;

        std::cout << "\nServer is running. Press Ctrl+C to stop." << std::endl;

        // Keep the main thread alive while the server runs
        while (running) {
            std::this_thread::sleep_for(std::chrono::milliseconds(100));
        }

        std::cout << "Stopping OPC UA server..." << std::endl;
        server.stop();

    } catch (const std::exception &e) {
        std::cerr << "Error: " << e.what() << std::endl;
        return 1;
    }

    return 0;
}
