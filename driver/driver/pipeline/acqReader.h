//
// Created by Synnax on 2/5/2024.
//

#pragma once
#include "synnax/synnax.h"
#include <atomic>
#include <memory>
#include <thread>
#
namespace daq {
    class AcqReader {
    public:
//        std::vector<long> time_index;
        std::vector<std::vector<long>> data;
        virtual std::pair <synnax::Frame, freighter::Error> read() = 0;
//        virtual freighter::Error dig italWrite(std::vector<std::uint64_t> channels, std::vector<std::uint64_t> values) = 0;
        virtual freighter::Error configure(synnax::Module config) = 0;
        virtual freighter::Error start() = 0;
        virtual freighter::Error stop() = 0;
    };

}
