/*
 * Raw pcap debug tool to diagnose EtherCAT communication.
 * This bypasses SOEM to see exactly what pcap sends/receives.
 */

#include <cstdio>
#include <cstring>
#include <pcap/pcap.h>
#include <unistd.h>

// EtherCAT ethertype
#define ETH_P_ECAT 0x88A4

// Simple EtherCAT frame structure
#pragma pack(push, 1)
struct EthHeader {
    uint8_t dest[6];
    uint8_t src[6];
    uint16_t ethertype;
};

struct EcatHeader {
    uint16_t length;  // 11 bits length + 1 bit reserved + 4 bits type
};

struct EcatDatagram {
    uint8_t cmd;
    uint8_t idx;
    uint16_t adp;
    uint16_t ado;
    uint16_t len;  // 11 bits length + 3 bits reserved + 1 bit circulating + 1 bit more
    uint16_t irq;
    // data follows, then 2-byte WKC
};
#pragma pack(pop)

void dump_hex(const uint8_t *data, int len) {
    for (int i = 0; i < len; i++) {
        printf("%02X ", data[i]);
        if ((i + 1) % 16 == 0) printf("\n");
    }
    if (len % 16 != 0) printf("\n");
}

void dump_mac(const uint8_t *mac) {
    printf("%02X:%02X:%02X:%02X:%02X:%02X",
           mac[0], mac[1], mac[2], mac[3], mac[4], mac[5]);
}

int main(int argc, char *argv[]) {
    if (argc < 2) {
        printf("Usage: %s <interface>\n", argv[0]);
        return 1;
    }

    const char *ifname = argv[1];
    char errbuf[PCAP_ERRBUF_SIZE];

    printf("Opening interface %s for raw pcap...\n", ifname);

    // Open pcap
    pcap_t *handle = pcap_open_live(ifname, 65535, 1, 100, errbuf);
    if (handle == NULL) {
        printf("ERROR: pcap_open_live failed: %s\n", errbuf);
        return 1;
    }

    // Set non-blocking
    if (pcap_setnonblock(handle, 1, errbuf) == -1) {
        printf("ERROR: pcap_setnonblock failed: %s\n", errbuf);
        pcap_close(handle);
        return 1;
    }

    // Set BPF filter for EtherCAT
    struct bpf_program fp;
    if (pcap_compile(handle, &fp, "ether proto 0x88a4", 1, PCAP_NETMASK_UNKNOWN) == -1) {
        printf("ERROR: pcap_compile failed\n");
        pcap_close(handle);
        return 1;
    }
    if (pcap_setfilter(handle, &fp) == -1) {
        printf("ERROR: pcap_setfilter failed\n");
        pcap_freecode(&fp);
        pcap_close(handle);
        return 1;
    }
    pcap_freecode(&fp);

    printf("pcap initialized successfully.\n\n");

    // Build a simple BRD (Broadcast Read) frame
    // BRD reads from all slaves, each slave increments WKC
    uint8_t frame[64];
    memset(frame, 0, sizeof(frame));

    // Ethernet header
    EthHeader *eth = (EthHeader *)frame;
    memset(eth->dest, 0xFF, 6);  // Broadcast
    eth->src[0] = 0x00;
    eth->src[1] = 0x02;
    eth->src[2] = 0x00;
    eth->src[3] = 0x00;
    eth->src[4] = 0x00;
    eth->src[5] = 0x01;
    eth->ethertype = htons(ETH_P_ECAT);

    // EtherCAT header (after Ethernet header, at offset 14)
    EcatHeader *ecat = (EcatHeader *)(frame + 14);
    // Length = 12 bytes (datagram header 10 + data 0 + WKC 2), Type = 1
    // Format: 11 bits length | 1 bit reserved | 4 bits type
    uint16_t ecat_len = 12;  // datagram size including WKC
    ecat->length = (ecat_len & 0x07FF) | (0x1 << 12);  // Type 1 = EtherCAT command

    // EtherCAT datagram (after EtherCAT header, at offset 16)
    EcatDatagram *dg = (EcatDatagram *)(frame + 16);
    dg->cmd = 0x07;  // BRD = Broadcast Read
    dg->idx = 0x01;  // Frame index
    dg->adp = 0x0000;  // Auto-increment address (0 for broadcast)
    dg->ado = 0x0130;  // Register: AL Status
    dg->len = (2 & 0x07FF);  // Read 2 bytes, no more datagrams
    dg->irq = 0x0000;

    // Data area (2 bytes for AL Status) + WKC (2 bytes) = 4 bytes at offset 26
    // frame[26], frame[27] = data (will be filled by slaves)
    // frame[28], frame[29] = WKC (starts at 0)

    int frame_len = 30;  // 14 (eth) + 2 (ecat hdr) + 10 (datagram hdr) + 2 (data) + 2 (wkc)

    printf("=== Sending EtherCAT BRD frame ===\n");
    printf("Frame length: %d bytes\n", frame_len);
    printf("Dest MAC: "); dump_mac(eth->dest); printf("\n");
    printf("Src MAC:  "); dump_mac(eth->src); printf("\n");
    printf("Frame hex:\n");
    dump_hex(frame, frame_len);
    printf("\n");

    // Send the frame
    if (pcap_sendpacket(handle, frame, frame_len) != 0) {
        printf("ERROR: pcap_sendpacket failed: %s\n", pcap_geterr(handle));
        pcap_close(handle);
        return 1;
    }
    printf("Frame sent successfully.\n\n");

    // Now receive frames for a bit
    printf("=== Receiving frames (waiting up to 1 second) ===\n");

    struct pcap_pkthdr *header;
    const uint8_t *pkt_data;
    int pkt_count = 0;

    for (int i = 0; i < 100; i++) {  // 100 x 10ms = 1 second
        int res = pcap_next_ex(handle, &header, &pkt_data);

        if (res == 1) {
            pkt_count++;
            printf("\n--- Packet %d received ---\n", pkt_count);
            printf("Length: %d bytes\n", header->caplen);

            if (header->caplen >= 14) {
                EthHeader *rx_eth = (EthHeader *)pkt_data;
                printf("Dest MAC: "); dump_mac(rx_eth->dest); printf("\n");
                printf("Src MAC:  "); dump_mac(rx_eth->src); printf("\n");
                printf("Ethertype: 0x%04X\n", ntohs(rx_eth->ethertype));

                // Check if it's our own frame or a response
                if (memcmp(rx_eth->src, eth->src, 6) == 0) {
                    printf(">>> This is OUR OWN frame (TX echo)\n");
                } else {
                    printf(">>> This is a RESPONSE from another device!\n");
                }
            }

            printf("Hex dump:\n");
            dump_hex(pkt_data, header->caplen);

            // Parse WKC if it's an EtherCAT frame
            if (header->caplen >= 30) {
                // WKC is at offset 28-29 for our simple frame
                uint16_t wkc = pkt_data[28] | (pkt_data[29] << 8);
                printf("WKC at offset 28: %d\n", wkc);
            }
        } else if (res == 0) {
            // Timeout, no packet
        } else if (res == -1) {
            printf("ERROR: pcap_next_ex failed: %s\n", pcap_geterr(handle));
            break;
        }

        usleep(10000);  // 10ms
    }

    printf("\n=== Summary ===\n");
    printf("Total packets received: %d\n", pkt_count);

    if (pkt_count == 0) {
        printf("\nNo packets received! Possible issues:\n");
        printf("  - BPF filter might be blocking our own TX\n");
        printf("  - No EtherCAT device responding\n");
        printf("  - pcap receive not working properly\n");
    } else if (pkt_count == 1) {
        printf("\nOnly 1 packet - likely just our TX echo, no response from device.\n");
    } else {
        printf("\nMultiple packets - check if any have different src MAC (= real response)\n");
    }

    pcap_close(handle);
    return 0;
}
