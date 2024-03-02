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
    class AcqReader //TODD: change to daqReader
    {
    public:
        std::vector<std::vector<long>> data;
        virtual std::pair<synnax::Frame, freighter::Error> read() = 0;
        virtual freighter::Error configure(synnax::Module config) = 0; // TODO: remove?
        virtual freighter::Error start() = 0;
        virtual freighter::Error stop() = 0;
    };

    class daqWriter{
    public:
        virtual std::pair<synnax::Frame, freighter::Error> write(synnax::Frame frame) = 0;
        virtual freighter::Error start() = 0;
        virtual freighter::Error stop() = 0;
        virtual freighter::Error configure(synnax::Module config) = 0; // TODO: remove?
        // other members
        // a structure to store errors?
    };


}
