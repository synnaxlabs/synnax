//
// Created by Synnax on 2/5/2024.
//

#pragma once
#include "synnax/synnax.h"
#include <atomic>
#include <memory>
#include <thread>

namespace daq
{
    class AcqReader
    {
    public:
        std::vector<std::vector<long>> data;
        virtual std::pair<synnax::Frame, freighter::Error> read() = 0;
        virtual freighter::Error configure(synnax::Module config) = 0;
        virtual freighter::Error start() = 0;
        virtual freighter::Error stop() = 0;
    };

}
