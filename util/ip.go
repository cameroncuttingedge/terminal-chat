package util

import "net"

// Helper function to get the local IP address of the server
func GetLocalIP() string {
    addrs, err := net.InterfaceAddrs()
    if err != nil {
        return "Unable to determine local IP"
    }
    for _, address := range addrs {
        if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
            if ipnet.IP.To4() != nil {
                return ipnet.IP.String()
            }
        }
    }
    return "No IP found"
}