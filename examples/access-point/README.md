# Access point example

This example starts a password-protected Wi-Fi access point at `192.168.4.1`,
assigns addresses to clients with DHCP, responds to pings, and logs to the USB serial monitor.

Flash it to a Pico W:

```shell
tinygo flash -target=pico -stack-size=8kb -scheduler=tasks -monitor ./examples/access-point
```

`StartAP` itself configures only the Wi-Fi link. The example separately attaches
an `lneto` network stack, assigns the Pico a static address, and runs the DHCP
server that lets devices finish connecting.
