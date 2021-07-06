## Sample URL hasher

A simple URL hasher. Fetches each URL and hashes the content with md5. 
* Results are printed to stdout.
* Errors (if any) are printed to stderr.
* Supports graceful termination via ^C.
* Timeout for each individual operation: one minute.
* Adds `http://` to URLs if needed.

### Installation:
* Install [go](https://golang.org/doc/install)
* Set up `GOBIN`, make sure it's in your `PATH`
* `go install -v github.com/derlaft/sample-url-hasher/cmd/sample-url-hasher@latest`

```
$ sample-url-hasher -help
Usage of sample-url-hasher:
  -parallel int
        Maximum number of parallel workers (default 10)
```

### Example:

```
$ sample-url-hasher example.com sportloto.ru
http://example.com 84238dfc8092e5d9c0dac8ef93371a07
2021/07/06 20:33:06 Error, could not fetch url (http://sportloto.ru): could not perform HTTP request: Get "https://sportloto.ru/": x509: certificate signed by unknown authority

$ sample-url-hasher -parallel 1 example.com sportloto.ru meowr.ru der.ttyh.ru habr.com ya.ru 2>/dev/null
http://example.com 84238dfc8092e5d9c0dac8ef93371a07
http://meowr.ru 739f804512e70eb4d15bd136a0b543ec
http://der.ttyh.ru 4da58e5cc77418db79bc9a7ea83f1cad
http://habr.com f7f51d34b333a35c9686df601a305e44
http://ya.ru 42a43a37decbc1e49fd16449e669741b
```
