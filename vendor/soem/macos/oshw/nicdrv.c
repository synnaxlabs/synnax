/*
 * macOS NIC driver for SOEM using pcap
 *
 * Key differences from Linux:
 * - Uses pcap instead of raw sockets (macOS doesn't have PF_PACKET)
 * - Filters TX echoes (macOS pcap sees our own transmitted frames)
 *
 * TX Echo Fix:
 * On macOS, pcap receives our own TX frames echoed back. We filter these by
 * checking if the source MAC exactly matches our primary/secondary MAC.
 * Real responses from EtherCAT slaves will have a modified source MAC.
 */

#include <sys/types.h>
#include <sys/ioctl.h>
#include <net/if.h>
#include <sys/socket.h>
#include <unistd.h>
#include <sys/time.h>
#include <time.h>
#include <arpa/inet.h>
#include <string.h>
#include <pthread.h>
#include <pcap.h>

#include "oshw.h"
#include "osal.h"

/** Redundancy modes */
enum
{
   /** No redundancy, single NIC mode */
   ECT_RED_NONE,
   /** Double redundant NIC connection */
   ECT_RED_DOUBLE
};

/** Primary source MAC address used for EtherCAT.
 * This address is not the MAC address used from the NIC.
 * EtherCAT does not care about MAC addressing, but it is used here to
 * differentiate the route the packet traverses through the EtherCAT
 * segment. This is needed to find out the packet flow in redundant
 * configurations. */
const uint16 priMAC[3] = EC_PRIMARY_MAC_ARRAY;
/** Secondary source MAC address used for EtherCAT. */
const uint16 secMAC[3] = EC_SECONDARY_MAC_ARRAY;

/** second MAC word is used for identification */
#define RX_PRIM priMAC[1]
/** second MAC word is used for identification */
#define RX_SEC  secMAC[1]

static void ecx_clear_rxbufstat(int *rxbufstat)
{
   int i;
   for (i = 0; i < EC_MAXBUF; i++)
   {
      rxbufstat[i] = EC_BUF_EMPTY;
   }
}

/** Basic setup to connect NIC to pcap.
 * @param[in] port        = port context struct
 * @param[in] ifname      = Name of NIC device, f.e. "en0"
 * @param[in] secondary   = if >0 then use secondary stack instead of primary
 * @return >0 if succeeded
 */
int ecx_setupnic(ecx_portt *port, const char *ifname, int secondary)
{
   int i;
   char errbuf[PCAP_ERRBUF_SIZE];
   pcap_t **ppcap;
   pthread_mutexattr_t mutexattr;

   if (secondary)
   {
      /* secondary port struct available? */
      if (port->redport)
      {
         /* when using secondary socket it is automatically a redundant setup */
         ppcap = &(port->redport->pcap_handle);
         *ppcap = NULL;
         port->redstate = ECT_RED_DOUBLE;
         port->redport->stack.pcap_handle = &(port->redport->pcap_handle);
         port->redport->stack.txbuf = &(port->txbuf);
         port->redport->stack.txbuflength = &(port->txbuflength);
         port->redport->stack.tempbuf = &(port->redport->tempinbuf);
         port->redport->stack.rxbuf = &(port->redport->rxbuf);
         port->redport->stack.rxbufstat = &(port->redport->rxbufstat);
         port->redport->stack.rxsa = &(port->redport->rxsa);
         ecx_clear_rxbufstat(&(port->redport->rxbufstat[0]));
      }
      else
      {
         /* fail */
         return 0;
      }
   }
   else
   {
      pthread_mutexattr_init(&mutexattr);
      pthread_mutex_init(&(port->getindex_mutex), &mutexattr);
      pthread_mutex_init(&(port->tx_mutex), &mutexattr);
      pthread_mutex_init(&(port->rx_mutex), &mutexattr);
      port->pcap_handle = NULL;
      port->lastidx = 0;
      port->redstate = ECT_RED_NONE;
      port->stack.pcap_handle = &(port->pcap_handle);
      port->stack.txbuf = &(port->txbuf);
      port->stack.txbuflength = &(port->txbuflength);
      port->stack.tempbuf = &(port->tempinbuf);
      port->stack.rxbuf = &(port->rxbuf);
      port->stack.rxbufstat = &(port->rxbufstat);
      port->stack.rxsa = &(port->rxsa);
      ecx_clear_rxbufstat(&(port->rxbufstat[0]));
      ppcap = &(port->pcap_handle);
   }

   /* Open pcap handle for the interface */
   *ppcap = pcap_open_live(ifname, 65536, 1, 1, errbuf);
   if (*ppcap == NULL)
   {
      return 0;
   }

   /* Set non-blocking mode */
   if (pcap_setnonblock(*ppcap, 1, errbuf) == -1)
   {
      pcap_close(*ppcap);
      *ppcap = NULL;
      return 0;
   }

   /* Note: pcap_set_immediate_mode() requires an inactive handle (from pcap_create),
    * but pcap_open_live() returns an already-activated handle. Calling it here
    * would be undefined behavior on macOS. Instead, we use a short read timeout
    * (1ms in pcap_open_live above) which provides similar low-latency behavior. */

   /* Setup ethernet headers in tx buffers so we don't have to repeat it */
   for (i = 0; i < EC_MAXBUF; i++)
   {
      ec_setupheader(&(port->txbuf[i]));
      port->rxbufstat[i] = EC_BUF_EMPTY;
   }
   ec_setupheader(&(port->txbuf2));

   return 1;
}

/** Close sockets used
 * @param[in] port        = port context struct
 * @return 0
 */
int ecx_closenic(ecx_portt *port)
{
   if (port->pcap_handle != NULL)
   {
      pcap_close(port->pcap_handle);
      port->pcap_handle = NULL;
   }
   if ((port->redport) && (port->redport->pcap_handle != NULL))
   {
      pcap_close(port->redport->pcap_handle);
      port->redport->pcap_handle = NULL;
   }

   return 0;
}

/** Fill buffer with ethernet header structure.
 * Destination MAC is always broadcast.
 * Ethertype is always ETH_P_ECAT.
 * @param[out] p = buffer
 */
void ec_setupheader(void *p)
{
   ec_etherheadert *bp;
   bp = p;
   bp->da0 = htons(0xffff);
   bp->da1 = htons(0xffff);
   bp->da2 = htons(0xffff);
   bp->sa0 = htons(priMAC[0]);
   bp->sa1 = htons(priMAC[1]);
   bp->sa2 = htons(priMAC[2]);
   bp->etype = htons(ETH_P_ECAT);
}

/** Get new frame identifier index and allocate corresponding rx buffer.
 * @param[in] port        = port context struct
 * @return new index.
 */
uint8 ecx_getindex(ecx_portt *port)
{
   uint8 idx;
   uint8 cnt;

   pthread_mutex_lock(&(port->getindex_mutex));

   idx = port->lastidx + 1;
   /* index can't be larger than buffer array */
   if (idx >= EC_MAXBUF)
   {
      idx = 0;
   }
   cnt = 0;
   /* try to find unused index */
   while ((port->rxbufstat[idx] != EC_BUF_EMPTY) && (cnt < EC_MAXBUF))
   {
      idx++;
      cnt++;
      if (idx >= EC_MAXBUF)
      {
         idx = 0;
      }
   }
   port->rxbufstat[idx] = EC_BUF_ALLOC;
   if (port->redstate != ECT_RED_NONE)
      port->redport->rxbufstat[idx] = EC_BUF_ALLOC;
   port->lastidx = idx;

   pthread_mutex_unlock(&(port->getindex_mutex));

   return idx;
}

/** Set rx buffer status.
 * @param[in] port        = port context struct
 * @param[in] idx      = index in buffer array
 * @param[in] bufstat  = status to set
 */
void ecx_setbufstat(ecx_portt *port, uint8 idx, int bufstat)
{
   port->rxbufstat[idx] = bufstat;
   if (port->redstate != ECT_RED_NONE)
      port->redport->rxbufstat[idx] = bufstat;
}

/** Transmit buffer over pcap (non blocking).
 * @param[in] port        = port context struct
 * @param[in] idx         = index in tx buffer array
 * @param[in] stacknumber  = 0=Primary 1=Secondary stack
 * @return socket send result
 */
int ecx_outframe(ecx_portt *port, uint8 idx, int stacknumber)
{
   int lp, rval;
   ec_stackT *stack;

   if (!stacknumber)
   {
      stack = &(port->stack);
   }
   else
   {
      stack = &(port->redport->stack);
   }
   lp = (*stack->txbuflength)[idx];
   (*stack->rxbufstat)[idx] = EC_BUF_TX;
   rval = pcap_inject(*stack->pcap_handle, (*stack->txbuf)[idx], lp);
   if (rval == -1)
   {
      (*stack->rxbufstat)[idx] = EC_BUF_EMPTY;
   }

   return rval;
}

/** Transmit buffer over pcap (non blocking).
 * @param[in] port        = port context struct
 * @param[in] idx = index in tx buffer array
 * @return socket send result
 */
int ecx_outframe_red(ecx_portt *port, uint8 idx)
{
   ec_comt *datagramP;
   ec_etherheadert *ehp;
   int rval;

   ehp = (ec_etherheadert *)&(port->txbuf[idx]);
   /* rewrite MAC source address 1 to primary */
   ehp->sa1 = htons(priMAC[1]);
   /* transmit over primary socket*/
   rval = ecx_outframe(port, idx, 0);
   if (port->redstate != ECT_RED_NONE)
   {
      pthread_mutex_lock(&(port->tx_mutex));
      ehp = (ec_etherheadert *)&(port->txbuf2);
      /* use dummy frame for secondary socket transmit (BRD) */
      datagramP = (ec_comt *)&(port->txbuf2[ETH_HEADERSIZE]);
      /* write index to frame */
      datagramP->index = idx;
      /* rewrite MAC source address 1 to secondary */
      ehp->sa1 = htons(secMAC[1]);
      /* transmit over secondary socket */
      port->redport->rxbufstat[idx] = EC_BUF_TX;
      if (pcap_inject(port->redport->pcap_handle, &(port->txbuf2), port->txbuflength2) == -1)
      {
         port->redport->rxbufstat[idx] = EC_BUF_EMPTY;
      }
      pthread_mutex_unlock(&(port->tx_mutex));
   }

   return rval;
}

/** Non blocking read of pcap. Put frame in temporary buffer.
 * @param[in] port        = port context struct
 * @param[in] stacknumber = 0=primary 1=secondary stack
 * @return >0 if frame is available and read
 */
static int ecx_recvpkt(ecx_portt *port, int stacknumber)
{
   int bytesrx;
   ec_stackT *stack;
   struct pcap_pkthdr *header;
   const u_char *pkt_data;
   int res;

   if (!stacknumber)
   {
      stack = &(port->stack);
   }
   else
   {
      stack = &(port->redport->stack);
   }

   res = pcap_next_ex(*stack->pcap_handle, &header, &pkt_data);
   if (res > 0)
   {
      bytesrx = header->caplen;
      if (bytesrx > sizeof(port->tempinbuf))
      {
         bytesrx = sizeof(port->tempinbuf);
      }
      memcpy(stack->tempbuf, pkt_data, bytesrx);
      port->tempinbufs = bytesrx;
      return 1;
   }

   return 0;
}

/** Non blocking receive frame function. Uses RX buffer and index to combine
 * read frame with transmitted frame. To compensate for received frames that
 * are out-of-order all frames are stored in their respective indexed buffer.
 * If a frame was placed in the buffer previously, the function retrieves it
 * from that buffer index without calling ec_recvpkt. If the requested index
 * is not already in the buffer it calls ec_recvpkt to fetch it. There are
 * three options now, 1 no frame read, so exit. 2 frame read but other
 * than requested index, store in buffer and exit. 3 frame read with matching
 * index, store in buffer, set completed flag in buffer status and exit.
 *
 * @param[in] port        = port context struct
 * @param[in] idx         = requested index of frame
 * @param[in] stacknumber = 0=primary 1=secondary stack
 * @return Workcounter if a frame is found with corresponding index, otherwise
 * EC_NOFRAME or EC_OTHERFRAME.
 */
int ecx_inframe(ecx_portt *port, uint8 idx, int stacknumber)
{
   uint16 l;
   int rval;
   uint8 idxf;
   ec_etherheadert *ehp;
   ec_comt *ecp;
   ec_stackT *stack;
   ec_bufT *rxbuf;

   if (!stacknumber)
   {
      stack = &(port->stack);
   }
   else
   {
      stack = &(port->redport->stack);
   }
   rval = EC_NOFRAME;
   rxbuf = &(*stack->rxbuf)[idx];
   /* check if requested index is already in buffer ? */
   if ((idx < EC_MAXBUF) && ((*stack->rxbufstat)[idx] == EC_BUF_RCVD))
   {
      l = (*rxbuf)[0] + ((uint16)((*rxbuf)[1] & 0x0f) << 8);
      /* return WKC */
      rval = ((*rxbuf)[l] + ((uint16)(*rxbuf)[l + 1] << 8));
      /* mark as completed */
      (*stack->rxbufstat)[idx] = EC_BUF_COMPLETE;
   }
   else
   {
      pthread_mutex_lock(&(port->rx_mutex));
      /* check again if requested index is already in buffer ?
       * other task might have received it before we grabbed mutex */
      if ((idx < EC_MAXBUF) && ((*stack->rxbufstat)[idx] == EC_BUF_RCVD))
      {
         l = (*rxbuf)[0] + ((uint16)((*rxbuf)[1] & 0x0f) << 8);
         /* return WKC */
         rval = ((*rxbuf)[l] + ((uint16)(*rxbuf)[l + 1] << 8));
         /* mark as completed */
         (*stack->rxbufstat)[idx] = EC_BUF_COMPLETE;
      }
      /* non blocking call to retrieve frame from pcap */
      else if (ecx_recvpkt(port, stacknumber))
      {
         rval = EC_OTHERFRAME;
         ehp = (ec_etherheadert *)(stack->tempbuf);

         /* MACOS TX ECHO FIX:
          * On macOS, pcap sees our own TX frames echoed back.
          * Skip frames that have our exact source MAC - these are TX echoes.
          * Real responses from EtherCAT slaves will have a modified MAC
          * (slaves modify the source MAC as the frame traverses the ring). */
         if ((ehp->sa0 == htons(priMAC[0])) &&
             (ehp->sa1 == htons(priMAC[1])) &&
             (ehp->sa2 == htons(priMAC[2])))
         {
            /* This is our own TX frame echoed back, skip it */
            pthread_mutex_unlock(&(port->rx_mutex));
            return EC_OTHERFRAME;
         }

         /* Also check secondary MAC if in redundant mode */
         if (port->redstate != ECT_RED_NONE)
         {
            if ((ehp->sa0 == htons(secMAC[0])) &&
                (ehp->sa1 == htons(secMAC[1])) &&
                (ehp->sa2 == htons(secMAC[2])))
            {
               /* This is our own TX frame echoed back, skip it */
               pthread_mutex_unlock(&(port->rx_mutex));
               return EC_OTHERFRAME;
            }
         }

         /* check if it is an EtherCAT frame */
         if (ehp->etype == htons(ETH_P_ECAT))
         {
            stack->rxcnt++;
            ecp = (ec_comt *)(&(*stack->tempbuf)[ETH_HEADERSIZE]);
            l = etohs(ecp->elength) & 0x0fff;
            idxf = ecp->index;
            /* found index equals requested index ? */
            if (idxf == idx)
            {
               /* yes, put it in the buffer array (strip ethernet header) */
               memcpy(rxbuf, &(*stack->tempbuf)[ETH_HEADERSIZE], (*stack->txbuflength)[idx] - ETH_HEADERSIZE);
               /* return WKC */
               rval = ((*rxbuf)[l] + ((uint16)((*rxbuf)[l + 1]) << 8));
               /* mark as completed */
               (*stack->rxbufstat)[idx] = EC_BUF_COMPLETE;
               /* store MAC source word 1 for redundant routing info */
               (*stack->rxsa)[idx] = ntohs(ehp->sa1);
            }
            else
            {
               /* check if index exist and someone is waiting for it */
               if (idxf < EC_MAXBUF && (*stack->rxbufstat)[idxf] == EC_BUF_TX)
               {
                  rxbuf = &(*stack->rxbuf)[idxf];
                  /* put it in the buffer array (strip ethernet header) */
                  memcpy(rxbuf, &(*stack->tempbuf)[ETH_HEADERSIZE], (*stack->txbuflength)[idxf] - ETH_HEADERSIZE);
                  /* mark as received */
                  (*stack->rxbufstat)[idxf] = EC_BUF_RCVD;
                  (*stack->rxsa)[idxf] = ntohs(ehp->sa1);
               }
               else
               {
                  /* strange things happened */
               }
            }
         }
      }
      pthread_mutex_unlock(&(port->rx_mutex));
   }

   /* WKC if matching frame found */
   return rval;
}

/** Blocking redundant receive frame function. If redundant mode is not active then
 * it skips the secondary stack and redundancy functions. In redundant mode it waits
 * for both (primary and secondary) frames to come in. The result goes in an decision
 * tree that decides, depending on the route of the packet and its possible missing arrival,
 * how to reroute the original packet to get the data in an other try.
 *
 * @param[in] port        = port context struct
 * @param[in] idx = requested index of frame
 * @param[in] timer = absolute timeout time
 * @return Workcounter if a frame is found with corresponding index, otherwise
 * EC_NOFRAME.
 */
static int ecx_waitinframe_red(ecx_portt *port, uint8 idx, osal_timert *timer)
{
   osal_timert timer2;
   int wkc = EC_NOFRAME;
   int wkc2 = EC_NOFRAME;
   int primrx, secrx;

   /* if not in redundant mode then always assume secondary is OK */
   if (port->redstate == ECT_RED_NONE)
      wkc2 = 0;

   do
   {
      /* only read frame if not already in */
      if (wkc <= EC_NOFRAME)
         wkc = ecx_inframe(port, idx, 0);
      /* only try secondary if in redundant mode */
      if (port->redstate != ECT_RED_NONE)
      {
         /* only read frame if not already in */
         if (wkc2 <= EC_NOFRAME)
            wkc2 = ecx_inframe(port, idx, 1);
      }
      /* Small delay to avoid busy polling */
      osal_usleep(1);
      /* wait for both frames to arrive or timeout */
   } while (((wkc <= EC_NOFRAME) || (wkc2 <= EC_NOFRAME)) && !osal_timer_is_expired(timer));

   /* only do redundant functions when in redundant mode */
   if (port->redstate != ECT_RED_NONE)
   {
      /* primrx if the received MAC source on primary socket */
      primrx = 0;
      if (wkc > EC_NOFRAME) primrx = port->rxsa[idx];
      /* secrx if the received MAC source on secondary socket */
      secrx = 0;
      if (wkc2 > EC_NOFRAME) secrx = port->redport->rxsa[idx];

      /* primary socket got secondary frame and secondary socket got primary frame */
      /* normal situation in redundant mode */
      if (((primrx == RX_SEC) && (secrx == RX_PRIM)))
      {
         /* copy secondary buffer to primary */
         memcpy(&(port->rxbuf[idx]), &(port->redport->rxbuf[idx]), port->txbuflength[idx] - ETH_HEADERSIZE);
         wkc = wkc2;
      }
      /* primary socket got nothing or primary frame, and secondary socket got secondary frame */
      /* we need to resend TX packet */
      if (((primrx == 0) && (secrx == RX_SEC)) ||
          ((primrx == RX_PRIM) && (secrx == RX_SEC)))
      {
         /* If both primary and secondary have partial connection retransmit the primary received
          * frame over the secondary socket. The result from the secondary received frame is a combined
          * frame that traversed all slaves in standard order. */
         if ((primrx == RX_PRIM) && (secrx == RX_SEC))
         {
            /* copy primary rx to tx buffer */
            memcpy(&(port->txbuf[idx][ETH_HEADERSIZE]), &(port->rxbuf[idx]), port->txbuflength[idx] - ETH_HEADERSIZE);
         }
         osal_timer_start(&timer2, EC_TIMEOUTRET);
         /* resend secondary tx */
         ecx_outframe(port, idx, 1);
         do
         {
            /* retrieve frame */
            wkc2 = ecx_inframe(port, idx, 1);
         } while ((wkc2 <= EC_NOFRAME) && !osal_timer_is_expired(&timer2));
         if (wkc2 > EC_NOFRAME)
         {
            /* copy secondary result to primary rx buffer */
            memcpy(&(port->rxbuf[idx]), &(port->redport->rxbuf[idx]), port->txbuflength[idx] - ETH_HEADERSIZE);
            wkc = wkc2;
         }
      }
   }

   /* return WKC or EC_NOFRAME */
   return wkc;
}

/** Blocking receive frame function. Calls ec_waitinframe_red().
 * @param[in] port        = port context struct
 * @param[in] idx       = requested index of frame
 * @param[in] timeout   = timeout in us
 * @return Workcounter if a frame is found with corresponding index, otherwise
 * EC_NOFRAME.
 */
int ecx_waitinframe(ecx_portt *port, uint8 idx, int timeout)
{
   int wkc;
   osal_timert timer;

   osal_timer_start(&timer, timeout);
   wkc = ecx_waitinframe_red(port, idx, &timer);

   return wkc;
}

/** Blocking send and receive frame function. Used for non processdata frames.
 * A datagram is build into a frame and transmitted via this function. It waits
 * for an answer and returns the workcounter. The function retries if time is
 * left and the result is WKC=0 or no frame received.
 *
 * The function calls ec_outframe_red() and ec_waitinframe_red().
 *
 * @param[in] port        = port context struct
 * @param[in] idx      = index of frame
 * @param[in] timeout  = timeout in us
 * @return Workcounter or EC_NOFRAME
 */
int ecx_srconfirm(ecx_portt *port, uint8 idx, int timeout)
{
   int wkc = EC_NOFRAME;
   osal_timert timer1, timer2;

   osal_timer_start(&timer1, timeout);
   do
   {
      /* tx frame on primary and if in redundant mode a dummy on secondary */
      ecx_outframe_red(port, idx);
      if (timeout < EC_TIMEOUTRET)
      {
         osal_timer_start(&timer2, timeout);
      }
      else
      {
         /* normally use partial timeout for rx */
         osal_timer_start(&timer2, EC_TIMEOUTRET);
      }
      /* get frame from primary or if in redundant mode possibly from secondary */
      wkc = ecx_waitinframe_red(port, idx, &timer2);
      /* wait for answer with WKC>=0 or otherwise retry until timeout */
   } while ((wkc <= EC_NOFRAME) && !osal_timer_is_expired(&timer1));

   return wkc;
}
