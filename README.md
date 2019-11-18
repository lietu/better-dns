[![License](https://img.shields.io/badge/License-BSD%203--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause)

# Better DNS

Local DNS based ad (etc) blocker inspired by [Pi-hole](https://pi-hole.net).

![Better DNS in action](./better-dns.gif)

What it can do:

 - Run locally, no need to host servers of any kind
 - Protect you in every network you're connected to
 - Block ads (and other unwanted things) in pretty much all programs you use (unless they use some custom DNS setup, which is not very common)
 - Parse common block lists from HTTP(S) urls
 - Block A & AAAA record resolution of addresses on those lists
 - Proxy any non-blacklisted DNS requests to a proper DNS server
 - Supports DNS-over-TLS with e.g. Cloudflare's `1.1.1.1:853` to avoid snooping
 - Override your active DNS servers while it's running and return them to normal on exit
 - Show all the DNS requests your software is doing - maybe you'll find it enlightening
 
What it can't do:

 - Protect other devices in your network, use [Pi-hole](https://pi-hole.net) for that

Current version is very preliminary, everything from blocklists to DNS servers is hard-coded. However, it seems to very much work (on Windows 10).

You probably have to run it as Administrator so it has enough permissions to edit your DNS server configuration.


## Future ideas

 - Performing DNS requests to multiple servers in parallel
 - Running as a service
 - Installers or similar
 - Support for Linux, Mac, maybe others (should be pretty easy to add)
 - Wide configuration options for e.g. DNS servers to use (UDP, TCP, DNS-over-TLS), interfaces to ignore, block lists, and custom blacklists
 - Cached block lists in case your network isn't working perfectly when you launch the software
 - Periodically checking the lists for updates (e.g. hourly / daily)
 - Support for DNS-over-HTTPS as well as DNS-over-TLS to bypass some filters
 - Monitor for new networks (e.g. WiFi) and update their DNS settings as well
 - Potentially caching of some results, though OSes and other things have their own DNS caches that are often annoying enough as it is so maybe at least an option to disable it
 - Logging of blocked requests
 - Reporting interface similar to Pi-hole


# License

Short answer: This software is licensed with the BSD 3-clause -license.

Long answer: The license for this software is in [LICENSE.md](./LICENSE.md), the libraries used may have varying other licenses that you need to be separately aware of.
