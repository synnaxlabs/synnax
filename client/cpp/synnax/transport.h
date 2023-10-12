#pragma once

#include "framer/framer.h"
#include "channel/channel.h"
#include "ranger/ranger.h"



class Transport {
public:
    Transport(uint16_t port, const std::string &ip);

    Framer::IteratorClient *frame_iter;
    Framer::StreamerClient *frame_stream;
    Framer::WriterClient *frame_write;
    Channel::CreateClient *chan_create;
    Channel::RetrieveClient *chan_retrieve;
    Ranger::RetrieveClient *range_retrieve;
    Ranger::CreateClient *range_create;
    Ranger::KVDeleteClient *range_kv_delete;
    Ranger::KVGetClient *range_kv_get;
    Ranger::KVSetClient *range_kv_set;
};
