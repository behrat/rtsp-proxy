# rtsp-proxy
rtsp-proxy is a golang program that allow public RTSP clients to connect to multiple RTSP servers on a private network behind one public IP address. It accepts for TCP connections, and tries to forward to the connection to the host specified in the RTSP request.

This only works if the network is using split-horizon DNS. Each desired RTSP server should have a unique public domain name, for which public DNS returns the public IP address of the server hosting rtsp-proxy. The rtsp-proxy itself uses an internal DNS server that resolves the same domain names to the individual RTSP server addresses on the private network.

For detailed information on the Real Time Streaming Protocol (RTSP) protocol, see the RFC 2326: https://www.ietf.org/rfc/rfc2326.txt

This server provides no security. The RTSP protocol supports HTTP-like authentication, which will be used if the RTSP server requests it, but everything is sent in plaintext. Many clients also support RTSP-over-HTTP as specified by Apple (http://www.opensource.apple.com/source/QuickTimeStreamingServer/QuickTimeStreamingServer-412.42/Documentation/RTSP_Over_HTTP.pdf), but the  clients I tested (VLC and LiveCams Pro on iOS) don't support using HTTPS.
