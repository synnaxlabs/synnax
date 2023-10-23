#include <string>
#include <vector>
#include <cstddef>
#include <vector>
#include <iostream>

//* Replace with actual telem and channel files
#include "telem.h" // Assuming that telem.h contains the definition of DataType
#include "channel.h" // Assuming that channel.h contains the definition of Keys

using namespace std;

//* Temporary Series/Frame implementation for Testing (Replace)
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
        channel::Keys keys; //number of keys in encoder

    public:
    //constructor
    EncoderDecoder(const std::vector<telem::DataType>& dataTypes, const channel::Keys& channelKeys)
        : dtypes(dataTypes), keys(channelKeys) {}

    unsigned char createFirstByte(bool equalDataSize, bool stronglyAlignedTimestampFlag, bool allChannels);
    std::vector<unsigned char> encode(Frame frame);
};


EncoderDecoder NewEncoderDecoder(const std::vector<telem::DataType>& DataTypes, const channel::Keys& ChannelKeys) {
    return EncoderDecoder{DataTypes, ChannelKeys};
}

    

    //Creating the First Byte 
    unsigned char EncoderDecoder::createFirstByte(bool equalDataSize, bool stronglyAlignedTimestampFlag, bool allChannels){
        unsigned char firstByte = 0;
        if (equalDataSize){
            firstByte |= (1 << 2);
        }
        if (stronglyAlignedTimestampFlag){
            firstByte |= (1 << 1);
        }
        if (allChannels) {
            firstByte |= 1;
        }
        return firstByte;
    }

    //Build and returnbyte array
    std::vector<unsigned char> EncoderDecoder::encode(Frame frame){

        bool sizeFlag = true;
        bool alignFlag = true;
        bool channelFlag = (keys.size() == frame.frameKeys.size());
        std::vector<unsigned char> byteArray;

        //Testing the characterstics so we can get the three flags

        // Initialize with the size of the first series' data
        uint32_t expectedDataSize = frame.series[0].Data.size(); 
        vector<uint64_t> expectedTimeRange = frame.series[0].Timerange;

        //Check if all data array sizes are same
        for (const auto& series : frame.series) {
            if (series.Data.size() != expectedDataSize){
                sizeFlag = false;
            }
        
        // Check if the time range is strongly/weakly aligned 
            if (series.Timerange != expectedTimeRange) {
                alignFlag = false;
            }
            
         }
        
        //Received all three flags

        //the first byte of each frame will contain flags for each array
        byteArray.push_back(createFirstByte(sizeFlag, alignFlag, channelFlag));
        
        //If equal_data_size_flag is True
            //next four bytes will include the size representing the size of all Series data arrays.

        if (sizeFlag) {
            uint32_t size = expectedDataSize;
            cout << "size: " << size << endl;

            for (int i = 0; i < 4; i++) {
                byteArray.push_back((size >> (i * 8)) & 0xFF);
            }

        }

        //If the Strongly Aligned Timestamp Flag is set to true, 
            //The following 16 bytes will include information about the timestamp for all Series arrays, with the startTime going first, and the endTime going second

        if (alignFlag) {
            uint64_t startTime = expectedTimeRange[0];
            uint64_t endTime = expectedTimeRange[1];
            std::cout << "startTime: " << startTime << endl;
            std::cout << "endTime: " << endTime << endl;

            for (int i = 0; i < 8; i++) {
                byteArray.push_back((startTime >> (i * 8)) & 0xFF);
            }
            for (int i = 0; i < 8; i++) {
                byteArray.push_back((endTime >> (i * 8)) & 0xFF);
            }
        }
        //If the allChannels is set to true, nothing will be different

        //for the rest of the byte array
        //Iterate through series sequential (0, 1, 2, ... n)
        //If Equal Data Size Flag is not set,
            // then the first four bytes should include the size of the data array
        //If All Channels Flag is not set,
            // then the next four bytes include the uint32 key for the designated series
            //Then, all values within the data array should be sent
            //If Strongly Aligned Timestamp Flag is not set, then the next 16 bytes should include timestamp information

        for (const auto& series : frame.series) {
            if (!sizeFlag) {
                int size = series.Data.size();

                for (int i = 0; i < 4; i++) {
                    byteArray.push_back((size >> (i * 8)) & 0xFF);
                }
            }

            if (!channelFlag) {
                uint32_t keyForCurrentSeries = frame.frameKeys[&series - &frame.series[0]];

                for (int i = 0; i < 4; i++) {
                    byteArray.push_back((keyForCurrentSeries >> (i * 8)) & 0xFF);
                }
            }
            //fix if working with floats (this currently assumes ints only)
            for (const auto& value : series.Data) {
                for (int i = 0; i < sizeof(value); i++) {
                    byteArray.push_back((value >> (i * 8)) & 0xFF);
                }
            }

            if (!alignFlag) {
                for (int i = 0; i < 8; i++) {
                    byteArray.push_back((series.Timerange[0] >> (i * 8)) & 0xFF);
                }

                 // Append the bytes of endTime
                for (int i = 0; i < 8; i++) {
                    byteArray.push_back((series.Timerange[1] >> (i * 8)) & 0xFF);
                }
            }
        }

    return byteArray;

    }
