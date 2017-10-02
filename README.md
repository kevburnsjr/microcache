microcache is an HTTP cache implemented as Go middleware.
Useful for APIs which serve large numbers of identical responses.
Especially useful in high traffic microservices to improve efficiency by
reducing read traffic through collapsed forwarding and improve availability
by serving stale responses should downstream services become unavailable.

## Notes

```
Separate middleware:
  Sanitize lang header? (first language)
  Sanitize region? (country code)

etag support?
if-modified-since support?
```

## Benchmarks
```
> gobench -u http://localhost/ -c 10 -t 10
Dispatching 10 clients
Waiting for results...

Requests:                           303459 hits
Successful requests:                303459 hits
Network failed:                          0 hits
Bad requests failed (!2xx):              0 hits
Successful requests rate:            30345 hits/sec
Read throughput:                 573895881 bytes/sec
Write throughput:                  2458115 bytes/sec
Test time:                              10 sec
```
