//Notes as of 11/7/2023: Series and Frame Implementation to be changed. EncoderDecoder completed.
//The current series and frame implementation was heavily influenced from the existing description on Git. 

#include <string>
#include <vector>
#include <cstddef>
#include <vector>
#include <iostream>

// Replace with the official Telem and Channel files
#include "telem.h" // Assuming that telem.h contains the definition of DataType
#include "channel.h" // Assuming that channel.h contains the definition of Keys


using namespace std;

// Temporary Series/Frame implementation (Replace)
template<typename T>
struct Series{
    telem::DataType Datatype; 
    std::vector<T> Data;
    std::vector<uint64_t> Timerange;
};

struct Frame{
    std::vector<std::uint32_t> frameKeys;
    std::vector<Series<int>> series;
};



class EncoderDecoder {
    private:
        std::vector<telem::DataType> dtypes;
        channel::Keys keys; //Number of keys in encoder

    public:
    EncoderDecoder(const std::vector<telem::DataType>& dataTypes, const channel::Keys& channelKeys)
        : dtypes(dataTypes), keys(channelKeys) {}

    unsigned char createFirstByte(bool equalDataSize, bool stronglyAlignedTimestampFlag, bool allChannels);
    std::vector<unsigned char> encode(Frame frame);
    Frame decode(const vector<unsigned char>& byteArray);
};


EncoderDecoder NewEncoderDecoder(const std::vector<telem::DataType>& DataTypes, const channel::Keys& ChannelKeys) {
    return EncoderDecoder{DataTypes, ChannelKeys};
}

    std::vector<unsigned char> EncoderDecoder::encode(Frame frame){

        bool sizeFlag = true;
        bool alignFlag = true;
        bool channelFlag = (keys.size() == frame.frameKeys.size());
        std::vector<unsigned char> byteArray;

        //Testing the characterstics to obtain the three flags

        // Initialize with the size of the first series' data
        uint32_t expectedDataSize = frame.series[0].Data.size(); 
        vector<uint64_t> expectedTimeRange = frame.series[0].Timerange;

        //Checking if all data array sizes are same
        for (const auto& series : frame.series) {
            if (series.Data.size() != expectedDataSize){
                sizeFlag = false;
            }
        
        // Checking if the time range is strongly/weakly aligned 
            if (series.Timerange != expectedTimeRange) {
                alignFlag = false;
            }
            
         }

        int cur_array_size = 1;
        
        //The first byte of each frame will contain flags
        unsigned char firstByte = 0;
        if (sizeFlag){
            firstByte |= (1 << 2);
        }
        if (alignFlag){
            firstByte |= (1 << 1);
        }
        if (channelFlag) {
            firstByte |= 1;
        }
        byteArray.push_back(firstByte);
        
        //If sizeFlag is True, the next four bytes will include the size representing the size of all Series data arrays.

        if (sizeFlag) {
            uint32_t size = expectedDataSize;
            //cout << "size: " << size << endl;
            byteArray.reserve(cur_array_size + 4);
            cur_array_size += 4;

            for (int i = 0; i < 4; i++) {
                byteArray.push_back((size >> (i * 8)) & 0xFF);
            }

        }

        //If alignFlag is true, the following 16 bytes will include information about the timestamp for all Series arrays, with the startTime going first, and the endTime going second

        if (alignFlag) {
            uint64_t startTime = expectedTimeRange[0];
            uint64_t endTime = expectedTimeRange[1];

            byteArray.reserve(cur_array_size + 16);
            cur_array_size += 16;

            for (int i = 0; i < 8; i++) {
                byteArray.push_back((startTime >> (i * 8)) & 0xFF);
            }
            for (int i = 0; i < 8; i++) {
                byteArray.push_back((endTime >> (i * 8)) & 0xFF);
            }
        }
        //Traversing through the data
        for (const auto& series : frame.series) {

            if (!channelFlag) {
                uint32_t keyForCurrentSeries = frame.frameKeys[&series - &frame.series[0]];
                byteArray.reserve(cur_array_size + 4);
                cur_array_size += 4;

                for (int i = 0; i < 4; i++) {
                    byteArray.push_back((keyForCurrentSeries >> (i * 8)) & 0xFF);
                }
            }
            if (!sizeFlag) {
                int size = series.Data.size();

                byteArray.reserve(cur_array_size + 4);
                cur_array_size += 4;

                for (int i = 0; i < 4; i++) {
                    byteArray.push_back((size >> (i * 8)) & 0xFF);
                }
            }
            //Note: Assumes working with ints only
            for (const auto& value : series.Data) {
                byteArray.reserve(cur_array_size + sizeof(value));
                cur_array_size += sizeof(value);
                for (int i = 0; i < sizeof(value); i++) {
                    byteArray.push_back((value >> (i * 8)) & 0xFF);
                }
            }

            if (!alignFlag) {
                byteArray.reserve(cur_array_size + 8);
                cur_array_size += 8;
                for (int i = 0; i < 8; i++) {
                    byteArray.push_back((series.Timerange[0] >> (i * 8)) & 0xFF);
                }

                 // Append the bytes of endTime
                 byteArray.reserve(cur_array_size + 8);
                cur_array_size += 8;
                for (int i = 0; i < 8; i++) {
                    byteArray.push_back((series.Timerange[1] >> (i * 8)) & 0xFF);
                }
            }
        }

    return byteArray;

    }

    Frame EncoderDecoder::decode(const vector<unsigned char>& byteArray) {
        Frame returnStruct;
        size_t index = 0;
        bool sizeFlag = byteArray[index] & (1 << 2);
        bool alignFlag = byteArray[index] & (1 << 1);
        bool channelFlag = byteArray[index] & 1;
        index += 1;

        //if Channel is true, simply copies EncoderDecoder's keys into the new Frame
        if (channelFlag) {
            returnStruct.frameKeys.reserve(keys.size());
            for (int i = 0; i < keys.size(); i++) {
                returnStruct.frameKeys.push_back(keys[i].get_value());
            }
        }

        uint32_t sizeRepresentation = 0;
        
        if (sizeFlag) {
            sizeRepresentation |= byteArray[index];
            sizeRepresentation |= static_cast<uint32_t>(byteArray[index + 1]) << 8;
            sizeRepresentation |= static_cast<uint32_t>(byteArray[index + 2]) << 16;
            sizeRepresentation |= static_cast<uint32_t>(byteArray[index + 3]) << 24;
        
            index += 4;
        }
    
        uint64_t timeRangeStart = 0;
        uint64_t timeRangeEnd = 0;

        if (alignFlag) {
            for (int i = 0; i < 8; i++) {
                timeRangeStart |= static_cast<uint64_t>(byteArray[index + i]) << (i * 8);
            }
            index += 8;
            for (int i = 0; i < 8; i++) {
                timeRangeEnd |= static_cast<uint64_t>(byteArray[index + i]) << (i * 8);
            }
            index += 8;
        }

        for (size_t i = 0; i < keys.size(); i++) {
        
            uint32_t seriesKey = 0;

            if (!channelFlag) {
                for (int i = 0; i < 4; i++) {
                    seriesKey |= static_cast<uint32_t>(byteArray[index + i]) << (i * 8);
                }
                if (seriesKey != keys[i].get_value()) {
                    continue;
                }
                returnStruct.frameKeys.push_back(seriesKey); 
                index += 4;
            }
        
            uint32_t curSize = 0;

            if (!sizeFlag) {
                for (int i = 0; i < 4; i++) {
                    curSize |= static_cast<uint32_t>(byteArray[index + i]) << (i * 8);
                }
                index += 4;
            }
            else {
                curSize = sizeRepresentation;
            }

            Series<int> curSeries; 
            curSeries.Data.resize(curSize);

        // Assuming byteArray is the input byte array
        for (int i = 0; i < curSize; i++) {
            uint32_t value = 0;
            for (int j = 0; j < sizeof(uint32_t); j++) {
                value |= static_cast<uint32_t>(byteArray[index]) << (j * 8);
                index++;
            }
            curSeries.Data[i] = value;
        }

        if (alignFlag == 0) {
            uint64_t start = 0;
            for (int j = 0; j < 8; j++) {
                start |= static_cast<uint64_t>(byteArray[index + j]) << (j * 8);
            }
            index += 8;

            uint64_t end = 0;
            for (int j = 0; j < 8; j++) {
                end |= static_cast<uint64_t>(byteArray[index + j]) << (j * 8);
            }
            index += 8;

            curSeries.Timerange = {start, end};
        } 
        else {
            curSeries.Timerange = {timeRangeStart, timeRangeEnd};
        }
    
        returnStruct.series.push_back(curSeries);
    }

    return returnStruct;

    }
