# prebuilt

A Go module that embeds a ready-made browser client for jargo bots, for the
jargo development runner to serve at `/client`:

```go
import "github.com/gojargo/jargo-web/prebuilt"

runner.Frontend = prebuilt.Handler()
```

The client is the published build of
[pipecat-ai-prebuilt](https://github.com/pipecat-ai/pipecat-prebuilt), embedded
unmodified (`prebuilt.Version` names the release). It connects over WebRTC, and
the other transports the runner serves, and shows the conversation, events and
metrics, which makes it a debugging UI rather than a product one.

To move to another release:

```sh
scripts/update-prebuilt.sh 1.3.0
```

The script downloads the release's wheel from PyPI, checks it against the
SHA-256 PyPI publishes, and copies its built client into `dist/`.

The client is distributed under the BSD 2-Clause License in [`LICENSE`](LICENSE),
copyright Daily; the notices for its own dependencies are in
[`dist/THIRD_PARTY_NOTICES.txt`](dist/THIRD_PARTY_NOTICES.txt).
