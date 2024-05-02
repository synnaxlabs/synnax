//
// Created by Synnax on 2/5/2024.
//

#pragma once

#include "client/cpp/synnax.h"
#include "driver/pipeline/acquisition.h"
#include "driver/pipeline/control.h"
#include <atomic>
#include <memory>
#include <thread>
#include "nlohmann/json.hpp" // for json parsing

using json = nlohmann::json;
namespace daq
{
    class daqReader : public pipeline::Source  //TODD: change to daqReader
    {
    public:
        virtual std::pair<synnax::Frame, freighter::Error> read() = 0;
        virtual freighter::Error start() = 0;
        virtual freighter::Error stop() = 0;
    };

    class daqWriter: public pipeline::Sink{
    public:
        virtual freighter::Error write(synnax::Frame frame) = 0;
        virtual freighter::Error start() = 0;
        virtual freighter::Error stop() = 0;
    };


    // class daqStateWriter : public pipeline::Source{
    // public:
    //     virtual std::pair<synnax::Frame, freighter::Error> read() = 0;
    //     virtual freighter::Error start() = 0;
    //     virtual freighter::Error stop() = 0;
    // };
}
